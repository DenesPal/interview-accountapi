// Copyleft 2020

// An example client for the Form3 toy-API exercise
package interview_accountapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// The default base URL for API root
	ApiBase = "https://api.form3.tech/"
	// Content-Type header to send when request has a JSON formatted body
	ContentType = "application/vnd.api+json"
	// Accept header to send with HTTP requests
	Accept = ContentType
	// Default overall request timeout
	DefaultTimeout = time.Duration(5) * time.Second
	// By default retry each HTTP request this many times if received an unexpected status code with the response
	DefaultRetries = 2
	// Delay between request-retries, calculated from the initiation of the previous request
	DefaultErrorBackOff = time.Duration(3) * time.Second
	// Number of items per page in paginated results (like List)
	DefaultPaginationSize = 100
	// Delay between (the initiation of) requests when iterating through the pages of a paginated response (like List)
	DefaultPaginationBackOff = time.Duration(400) * time.Millisecond
)

// The Form3 API client
type ApiClient struct {
	// Retry HTTP requests N times if received an unexpected status code, min 1
	Retries uint
	// Wait between initiation of requests in a retry scenario
	ErrorBackOff time.Duration
	// Wait between initiation of requests when iterating over the pages of a paginated response (like List)
	PaginationBackOff time.Duration
	// Base URL for API requests
	baseURL *url.URL
	// The underlying HTTP client
	httpClient *http.Client
	// Number of items per page for List actions (default 100, max 1000)
	pageSize uint
}

// NewApiClient creates a new Form3 API client with defaults
func NewApiClient() *ApiClient {
	client := ApiClient{
		Retries:           DefaultRetries,
		ErrorBackOff:      DefaultErrorBackOff,
		PaginationBackOff: DefaultPaginationBackOff,
		pageSize:          DefaultPaginationSize,
	}

	client.httpClient = &http.Client{Timeout: DefaultTimeout}

	e := client.SetBaseURL(ApiBase)
	if e != nil {
		log.Panicf("Failed parsing base URL constant: %s: %s", e, ApiBase)
	}

	return &client
}

// Gets client.pageSize
func (client *ApiClient) PageSize() int {
	return int(client.pageSize)
}

// Sets number of results requested per pagination page
func (client *ApiClient) SetPageSize(pageSize int) {
	if pageSize < 1 {
		client.pageSize = 1
	} else if pageSize > 1000 {
		client.pageSize = 1000
	} else {
		client.pageSize = uint(pageSize)
	}
}

// Gets current API root URL as string
func (client *ApiClient) BaseURL() string {
	return client.baseURL.String()
}

// SetBaseURL sets/changes the API root URL by parsing an URL string
func (client *ApiClient) SetBaseURL(apiBase string) error {
	var err error
	client.baseURL, err = url.Parse(apiBase)
	return err // returns error or nil if success
}

// Gets overall request timeout (time.Duration)
func (client *ApiClient) Timeout() time.Duration {
	return client.httpClient.Timeout
}

// Sets overall request timeout (time.Duration)
func (client *ApiClient) SetTimeout(duration time.Duration) {
	client.httpClient.Timeout = duration
}

// NewRequest creates a new HTTP request of method with path relative to the baseURL of the client,
// and an optional body of io.Reader or nil
func (client *ApiClient) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		log.Print("Failed parsing path,", err, path)
		return nil, err
	}
	u = client.baseURL.ResolveReference(u)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		log.Print("Failed creating", method, "request,", err, path)
		return nil, err
	}

	req.Header.Set("Accept", Accept)
	if body != nil {
		req.Header.Set("Content-Type", ContentType)
	}

	return req, nil
}

// Do executes the http.Request with timeout and Retries
//
// A response status code of >= 200 < 300 is considered successful.
//
// Status codes <200 400 401 403 404 405 406 407 409 410 414 418 431 are considered unrecoverable and not retried.
// Timeout is calculated from the initiation of the request.
// There is an ErrorBackOff delay between they initiation of Retries. When retries are exhausted,
// the error of the last request is returned.
//
// Retrying introduces a trade-off with POST (Create) requests as it may result in a Conflict on succeeding tries if
// the success from the first try got hidden. This shall be handled by the caller. (see CreateAccount for example)
//
// Returned ApiError has Error interface with StatusCode property with the returned HTTP status code.
// If an error message is present in the response, it is parsed
func (client *ApiClient) Do(req *http.Request) (*http.Response, *ApiError) {
	var body []byte
	var err error
	var resp *http.Response

	if req.Body != nil {
		// Reuse request body between retries
		// attribution: https://stackoverflow.com/a/54706278
		if body, err = ioutil.ReadAll(req.Body); err != nil {
			log.Panic("failed sucking up request body")
		}
		if err = req.Body.Close(); err != nil {
			log.Panic("failed closing request body")
		}
	}

	var lastTime time.Time
Retry:
	for turn := uint(0); turn < client.Retries; turn++ {
		if req.Body != nil {
			// Recreating request body for each requests
			req.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

		if turn > 0 {
			sleepDuration := client.ErrorBackOff - time.Now().Sub(lastTime)
			if sleepDuration > 0 {
				log.Printf("Retrying %s request in %v %s %s",
					req.Proto, sleepDuration, req.Method, req.URL.String())
				time.Sleep(sleepDuration)
			}
		}

		// Executes the actual HTTP request here
		log.Printf("%s request %s %s", req.Proto, req.Method, req.URL.String())
		lastTime = time.Now()
		resp, err = client.httpClient.Do(req)

		if err != nil {
			log.Printf("%s request failed: %s", req.Proto, err)
			continue Retry
		}

		if resp == nil {
			log.Panic("http.client.Do() returned nil request and nil error")
		}

		log.Printf("%s response %s %v (%d bytes) from %s %s",
			resp.Proto, resp.Status, time.Now().Sub(lastTime), resp.ContentLength,
			req.Method, req.URL.String())

		if 0 < resp.StatusCode && resp.StatusCode < 300 {
			// success (perhaps should be more strict <= 200)
			break Retry
		}

		// Some errors shan't be repeated
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusMethodNotAllowed, http.StatusNotAcceptable, http.StatusProxyAuthRequired,
			http.StatusConflict, http.StatusGone, http.StatusRequestURITooLong, http.StatusTeapot,
			http.StatusRequestHeaderFieldsTooLarge:
			break Retry
		}

		if e := resp.Body.Close(); e != nil {
			log.Print("Closing of response body failed!")
		}
	}

	var apiErr *ApiError
	if err != nil {
		apiErr = NewApiError(resp, err.Error())
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr = NewApiError(resp, "Received unexpected HTTP status code %s", resp.Status)
	}
	return resp, apiErr
}

// JsonRequest creates and executes an HTTP request of method with relative path to the baseURL and an optional data
// (or nil) in the request body (serializes it as JSON). Returns the HTTP response, the JSON decoder, and APIError.
func (client *ApiClient) JsonRequest(method string, path string, data interface{}) (
	*http.Response, *json.Decoder, *ApiError) {
	var (
		body *bytes.Reader
		err  error
		req  *http.Request
	)

	if data == nil {
		req, err = client.NewRequest(method, path, nil)

	} else {
		// Encode JSON data and present as io.Reader
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, nil, NewApiError(nil, err.Error())
		}
		body = bytes.NewReader(jsonData)

		req, err = client.NewRequest(method, path, body)
	}
	if err != nil {
		return nil, nil, NewApiError(nil, err.Error())
	}

	resp, apiErr := client.Do(req)
	if apiErr != nil {
		return resp, nil, apiErr
	}

	dec, err := decodeJsonResponse(resp)
	if err != nil {
		return resp, nil, NewApiError(nil, err.Error())
	}
	return resp, dec, nil
}

// decodeJsonResponse returns a JSON decoder if response had the expected Content-Type header.
func decodeJsonResponse(resp *http.Response) (*json.Decoder, error) {
	ctype := resp.Header["Content-Type"]
	if len(ctype) < 1 || strings.ToLower(ctype[0]) == ContentType {
		err := errors.New(fmt.Sprint("Received unknown Content-Type:", ctype))
		log.Print(err)
		return nil, err
	}
	return json.NewDecoder(resp.Body), nil
}
