package interview_accountapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"
)

const AccountsPath = "v1/organisation/accounts"

type AccountListResults struct {
	channel chan<- Account
	error   error
}

func (client *ApiClient) ListAccounts(filters map[string]string) (<-chan *Account, *ApiError) {
	var apiErr *ApiError
	accounts := make(chan *Account)

	// Append filters and pagination to query string //
	u, q, err := parseURL(AccountsPath)
	if err != nil {
		return nil, NewApiError(nil, err.Error())
	}

	q.Set("page[size]", fmt.Sprint(client.PageSize))

	for k, v := range filters {
		if !accountListFilters[k] {
			return nil, NewApiError(nil, "invalid filter key: %s", k)
		}
		q.Set(fmt.Sprintf("filter[%s]", k), v)
	}

	pth := assembleURL(u, q)

	// fixme implement a better way to return error, from subsequent loops too //
	go func() {
		var (
			dec      *json.Decoder
			resp     *http.Response
			lastTime time.Time
		)

		for i := 0; pth != ""; i++ {
			sleepDuration := client.PagingBackOff - time.Now().Sub(lastTime)
			if 0 < i && 0 < sleepDuration {
				time.Sleep(sleepDuration)
			}
			lastTime = time.Now()

			resp, dec, apiErr = client.JsonRequest(http.MethodGet, pth, nil)
			if apiErr != nil {
				break
			}

			var response AccountDetailsListResponse
			err := dec.Decode(&response)
			if e := resp.Body.Close(); e != nil {
				log.Print("Closing of response body failed!")
			}

			if err != nil {
				apiErr = NewApiError(resp, err.Error())
				break
			}

			// signals outer func to return Fixme better solution is needed //
			if i == 0 {
				accounts <- nil
			}

			for _, acc := range response.Data {
				accounts <- acc
			}

			if response.Links == nil {
				break
			}
			pth = response.Links.Next

		}
		close(accounts)
	}()

	// block until first result is ready,
	// also abusing a apiErr which is shared with internal goroutine to get at least the error of the first request
	<-accounts

	return accounts, apiErr
}

func (client *ApiClient) CreateAccount(account *Account) (*Account, *ApiError) {
	if err := account.Validate(); err != nil {
		return nil, NewApiError(nil, err.Error())
	}

	resp, dec, apiErr := client.JsonRequest(http.MethodPost, AccountsPath, AccountCreation{account})
	if apiErr != nil {
		return nil, apiErr
	}

	var response AccountCreationResponse
	if err := dec.Decode(&response); err != nil {
		apiErr = NewApiError(resp, err.Error())
	}
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, apiErr
}

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
