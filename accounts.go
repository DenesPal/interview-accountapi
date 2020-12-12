package interview_accountapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	path2 "path"
)

const AccountsPath = "v1/organisation/accounts"

type AccountListResults struct {
	channel chan<- Account
	error   error
}

func (client *ApiClient) ListAccounts(filters map[string]string) (<-chan *Account, error) {
	accounts := make(chan *Account)
	var (
		dec  *json.Decoder
		err  error
		resp *http.Response
	)

	// Append filters and pagination to query string //
	u, q, err := parseURL(AccountsPath)
	if err != nil {
		return nil, err
	}

	q.Set("page[size]", fmt.Sprint(client.PageSize))

	for k, v := range filters {
		if !accountListFilters[k] {
			return nil, errors.New(fmt.Sprint("invalid filter key", k))
		}
		q.Set(fmt.Sprintf("filter[%s]", k), v)
	}

	path := assembleURL(u, q)

	// fixme implement a better way to return error, from subsequent loops too //
	go func() {
		for i := 0; path != ""; i++ {
			resp, dec, err = client.JsonRequest(http.MethodGet, path, nil)
			if err != nil {
				close(accounts)
				return
			}

			var response AccountDetailsListResponse
			err = dec.Decode(&response)
			if e := resp.Body.Close(); e != nil {
				log.Print("Closing of response body failed!")
			}

			if err != nil {
				close(accounts)
				return
			}

			// signals outer func to return Fixme better solution is needed //
			if i == 0 {
				accounts <- nil
			}

			for _, acc := range response.Data {
				accounts <- acc
			}

			path = response.Links.Next

		}
		close(accounts)
	}()

	// block until first result is ready fixme find better solution //
	<-accounts

	return accounts, err
}

func (client *ApiClient) CreateAccount(account *Account) (*Account, error) {
	if err := account.Validate(); err != nil {
		return nil, err
	}

	resp, dec, err := client.JsonRequest(http.MethodPost, AccountsPath, AccountCreation{account})
	if err != nil {
		return nil, err
	}

	var response AccountCreationResponse
	err = dec.Decode(&response)
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, err
}

func (client *ApiClient) UpdateAccount(id string, account *Account) (*Account, error) {
	if id == "" {
		return nil, errors.New("empty account id")
	}
	path := path2.Join(AccountsPath, id)

	if err := account.Validate(); err != nil {
		return nil, err
	}

	resp, dec, err := client.JsonRequest(http.MethodPatch, path, AccountCreation{account})
	if err != nil {
		return nil, err
	}

	var response AccountDetailsResponse
	err = dec.Decode(&response)
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, err
}

func (client *ApiClient) FetchAccount(id string) (*Account, error) {
	if id == "" {
		return nil, errors.New("empty account id")
	}
	path := path2.Join(AccountsPath, id)

	resp, dec, err := client.JsonRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AccountDetailsResponse
	err = dec.Decode(&response)
	if e := resp.Body.Close(); e != nil {
		log.Print("Closing of response body failed!")
	}

	return response.Data, err
}

func (client *ApiClient) DeleteAccount(id string, version uint) error {
	if id == "" {
		return errors.New("empty account id")
	}

	u, v, err := parseURL(AccountsPath)
	if err != nil {
		return err
	}

	u.Path = path2.Join(u.Path, id)
	v.Set("version", fmt.Sprint(version))

	path := assembleURL(u, v)

	req, err := client.NewRequest(http.MethodDelete, path, nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.New(fmt.Sprintf("failed to delete account %s received status code %d", id, resp.StatusCode))
	}

	return nil
}
