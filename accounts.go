// Copyleft 2020

package interview_accountapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"
)

// Path to Account resources, relative to API root path
const AccountsPath = "v1/organisation/accounts"

// Results of ListAccounts
type AccountListResults struct {
	// Channel of Account resources, automatically iterating through pagination (unbuffered)
	Channel chan *Account
	// Error message (listing go-routine stops on error, stores it here, then closes the channel
	Error *ApiError
	// Used by Close to signal go-routine to terminate
	closing chan bool
}

// Closes AccountListResults: signals internal go-routine of ListAccounts to terminate.
//
// Never blocks because closing channel is buffered 1, also makes it safe to be invoked multiple times.
// Also need not wait for the cleanup of the internal go-routine.
//
// If consumer stops receiving exactly after the last item of a page, the retrieval of the next page can run
// beyond Close, as well as any errors related to that page remain hidden. This can be avoided making the closing
// channel unbuffered, but then Close in it's current form could potentially block,
// also could freeze if called multiple times.
func (results *AccountListResults) Close() {
	select {
	case results.closing <- true:
	default:
	}
}

// Sets AccountListResults.Error and closes Channel, used by ListAccounts on error or to terminate internal go-routine
func (results *AccountListResults) finish(apiErr *ApiError) {
	// nil if no error
	results.Error = apiErr
	// Signals finish to receivers by closing the channel
	close(results.Channel)
}

// List Account resources with optional filters (or nil), returns AccountListResults
//
// Parses one page at a time, feeding Account results through AccountListResults.Channel,
// and fetches next page when the last item of a page is consumed.
//
// On error sets AccountListResults.Error (type ApiError) then closes AccountListResults.Channel
//
// AccountListResults.Close shall be invoked to terminate the internal go-routine
//
// A possible use pattern is to iterate with range over the AccountListResults.Channel
// and check for AccountListResults.Error when the results are exhausted (since feeding stops on error).
func (client *ApiClient) ListAccounts(filters map[string]string) *AccountListResults {
	results := &AccountListResults{Channel: make(chan *Account), closing: make(chan bool, 1)}

	// Append filters and pagination to query string
	u, q, err := parseURL(AccountsPath)
	if err != nil {
		results.finish(NewApiError(nil, err.Error()))
		return results
	}

	if client.PageSize > 1000 {
		q.Set("page[size]", "1000")
	} else {
		q.Set("page[size]", fmt.Sprint(client.PageSize))
	}

	for k, v := range filters {
		if !accountListFilters[k] {
			results.finish(NewApiError(nil, "invalid filter key: %s", k))
			return results
		}
		q.Set(fmt.Sprintf("filter[%s]", k), v)
	}

	pth := assembleURL(u, q)

	// Internal go-routine to fetch successive pages and feed results to channel
	go func() {
		var (
			apiErr   *ApiError
			dec      *json.Decoder
			resp     *http.Response
			lastTime time.Time
		)

		for i := 0; pth != ""; i++ {
			// Waits between requesting successive pages
			sleepDuration := client.PagingBackOff - time.Now().Sub(lastTime)
			if 0 < i && 0 < sleepDuration {
				time.Sleep(sleepDuration)
			}
			lastTime = time.Now()

			// Does the actual HTTP request and returns a JSON decoder
			resp, dec, apiErr = client.JsonRequest(http.MethodGet, pth, nil)
			if apiErr != nil {
				break
			}

			// JSON-decodes response body
			var response AccountDetailsListResponse
			err := dec.Decode(&response)

			// Close response body (already read all)
			if e := resp.Body.Close(); e != nil {
				// Probably safe to ignore this error, hence it is only logged, but isn't propagated through the chan
				log.Printf("Closing of response body failed: %s", e)
			}

			// Stops on JSON decoding error from above
			if err != nil {
				apiErr = NewApiError(resp, err.Error())
				break
			}

			// Feeds results from current page to channel (one-by-one, blocking)
			for _, acc := range response.Data {
				select {
				case <-results.closing:
					// Stops on close message
					break
				case results.Channel <- acc:
				}
			}

			if response.Links == nil {
				// Was last page
				break
			}
			// Iterates to next page
			pth = response.Links.Next

		}

		// Exposes error (if any) and signals finish to receivers
		results.finish(apiErr)
	}()

	return results
}

// Creates an Account resource and returns the latest version of it
func (client *ApiClient) CreateAccount(account *Account) (*Account, *ApiError) {
	if err := account.Validate(); err != nil {
		return nil, NewApiError(nil, err.Error())
	}

	// Retrying a POST request can raise a 409 Conflict, this is a scrappy work-around part 1:
	// Check for existing resource by id and raise a Conflict error now. Then Conflict errors for the POST request
	// can be interpreted as a retry scenario where the success of the first try was lost.
	existing, apiErr := client.FetchAccount(account.Id)
	if apiErr == nil {
		apiErr = NewApiError(nil, "Account with id %s already exists", existing.Id)
		apiErr.StatusCode = http.StatusConflict
		return existing, apiErr
	} else if apiErr.StatusCode != http.StatusNotFound {
		return nil, apiErr
	}

	resp, dec, apiErr := client.JsonRequest(http.MethodPost, AccountsPath, AccountCreation{account})

	if apiErr == nil {
		var response AccountCreationResponse
		if err := dec.Decode(&response); err != nil {
			apiErr = NewApiError(resp, err.Error())
		}
		if e := resp.Body.Close(); e != nil {
			log.Print("Closing of response body failed!")
		}
		return response.Data, apiErr

	} else if resp != nil && resp.StatusCode == http.StatusConflict {
		// Work-around part 2: In case of Conflict, fetch and return the existing resource.
		// This would introduce a race condition if the same id was used to create resources across multiple clients.
		if latest, err := client.FetchAccount(account.Id); err == nil {
			return latest, nil
		}
	}

	return nil, apiErr
}

// Updates an Account resource, returns the resource as received in the response
func (client *ApiClient) UpdateAccount(id string, account *Account) (*Account, *ApiError) {
	if id == "" {
		return nil, NewApiError(nil, "Empty account id")
	}
	pth := path.Join(AccountsPath, id)

	if err := account.Validate(); err != nil {
		return nil, NewApiError(nil, err.Error())
	}

	resp, dec, apiErr := client.JsonRequest(http.MethodPatch, pth, AccountAmendment{account})
	if apiErr != nil {
		return nil, apiErr
	}

	var response AccountDetailsResponse
	if err := dec.Decode(&response); err != nil {
		apiErr = NewApiError(resp, err.Error())
	}
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, apiErr
}

// Fetches an Account resource by id, if missing, returns ApiError with .code as 404.
func (client *ApiClient) FetchAccount(id string) (*Account, *ApiError) {
	if id == "" {
		return nil, NewApiError(nil, "Empty account id")
	}
	pth := path.Join(AccountsPath, id)

	resp, dec, apiErr := client.JsonRequest(http.MethodGet, pth, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	var response AccountDetailsResponse
	if err := dec.Decode(&response); err != nil {
		apiErr = NewApiError(resp, err.Error())
	}
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, apiErr
}

// Deletes an Account resource by id, returns error or nil on success
func (client *ApiClient) DeleteAccount(id string, version uint) *ApiError {
	if id == "" {
		return NewApiError(nil, "Empty account id")
	}

	u, v, err := parseURL(AccountsPath)
	if err != nil {
		return NewApiError(nil, err.Error())
	}

	u.Path = path.Join(u.Path, id)
	v.Set("version", fmt.Sprint(version))

	pth := assembleURL(u, v)

	req, err := client.NewRequest(http.MethodDelete, pth, nil)

	resp, apiErr := client.Do(req)
	if apiErr != nil {
		return apiErr
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return NewApiError(resp, "Failed to delete account %s received status %s", id, resp.Status)
}
