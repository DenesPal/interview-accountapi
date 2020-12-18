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
	DefaultRetries = 3
	// Delay between request-retries, calculated from the initiation of the previous request
	DefaultErrorBackOff = time.Duration(3) * time.Second
	// Delay between (the initiation of) requests when iterating through the pages of a paginated response (like List)
	DefaultPagingBackOff = time.Duration(400) * time.Millisecond
)

// The Form3 API client
type ApiClient struct {
	// Base URL for API requests
	BaseURL *url.URL
	// Timeout of HTTP requests
	timeout time.Duration
	// Retry HTTP requests N times if received an unexpected status code
	Retries uint
	// Wait between initiation of requests in a retry scenario
	ErrorBackOff time.Duration
	// Wait between initiation of requests when iterating over the pages of a paginated response (like List)
	PagingBackOff time.Duration
	// The underlying HTTP client
	httpClient *http.Client
	// Number of items per page for List actions (default 100)
	PageSize uint
}

// NewApiClient creates a new Form3 API client with defaults
func NewApiClient() *ApiClient {
	u, e := url.Parse(ApiBase)
	if e != nil {
		log.Panicf("Failed parsing base URL constant: %s: %s", e, ApiBase)
	}
	client := ApiClient{
		BaseURL:       u,
		timeout:       DefaultTimeout,
		Retries:       DefaultRetries,
		ErrorBackOff:  DefaultErrorBackOff,
		PagingBackOff: DefaultPagingBackOff,
		PageSize:      100,
	}
	client.httpClient = &http.Client{Timeout: DefaultTimeout}

	return &client
}

// SetBaseURL sets/changes the API root URL by parsing an URL string
func (client *ApiClient) SetBaseURL(apiBase string) error {
	var err error
	client.BaseURL, err = url.Parse(apiBase)
	return err // returns error or nil if success
}

// NewRequest creates a new HTTP request of method with path relative to the BaseURL of the client,
// and an optional body of io.Reader or nil
func (client *ApiClient) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		log.Print("Failed parsing path,", err, path)
		return nil, err
	}
	u = client.BaseURL.ResolveReference(u)

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
// Status codes <200 400 401 403 404 405 406 407 410 414 418 431 are considered unrecoverable and not retried.
// Timeout is calculated from the initiation of the request.
// There is an ErrorBackOff delay between they initiation of Retries. When retries are exhausted,
// the error of the last request is returned.
//
// Returned ApiError has Error interface with Code property with the returned HTTP status code
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
				log.Printf("Retrying %s request in %v %s %s", req.Proto, sleepDuration, req.Method, req.URL.String())
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

		log.Printf("%s response %s (%d bytes) for %s %s",
			resp.Proto, resp.Status, resp.ContentLength, req.Method, req.URL.String())

		if 0 < resp.StatusCode && resp.StatusCode < 300 {
			// success (perhaps should be more strict <= 200)
			break Retry
		}

		// Some errors shan't be repeated
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound,
			http.StatusMethodNotAllowed, http.StatusNotAcceptable, http.StatusProxyAuthRequired, http.StatusGone,
			http.StatusRequestURITooLong, http.StatusTeapot, http.StatusRequestHeaderFieldsTooLarge:
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
		apiErr = NewApiError(resp, "Received HTTP status %s", resp.Status)
	}
	return resp, apiErr
}

// JsonRequest creates and executes an HTTP request of method with relative path to the BaseURL and an optional data
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
		return resp, nil, NewApiError(resp, err.Error())
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
