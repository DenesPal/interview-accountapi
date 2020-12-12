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
	ApiBase        = "https://api.form3.tech/"
	ContentType    = "application/vnd.api+json"
	Accept         = ContentType
	DefaultTimeout = time.Duration(5) * time.Second
	DefaultRetries = 3
	DefaultBackOff = time.Duration(3) * time.Second
)

type ApiClient struct {
	BaseURL    *url.URL
	Timeout    time.Duration
	Retries    uint
	BackOff    time.Duration
	HTTPClient *http.Client
	PageSize   uint
}

func NewClient(client ApiClient) (*ApiClient, error) {
	if client.Timeout == 0 {
		client.Timeout = DefaultTimeout
	}
	if client.Retries == 0 {
		client.Retries = DefaultRetries
	}
	if client.BackOff == 0 {
		client.BackOff = DefaultBackOff
	}
	if client.PageSize == 0 {
		client.PageSize = 100
	}

	if client.HTTPClient == nil {
		client.HTTPClient = &http.Client{Timeout: client.Timeout}
	}

	if client.BaseURL == nil {
		var err error
		client.BaseURL, err = url.Parse(ApiBase)
		if err != nil {
			log.Panic("Failed parsing base URL constant,", err, ApiBase)
		}
	}

	return &client, nil
}

func (client *ApiClient) SetBaseURL(apiBase string) error {
	var err error
	client.BaseURL, err = url.Parse(apiBase)
	return err // returns error or nil if success //
}

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

func (client *ApiClient) Do(req *http.Request) (*http.Response, error) {
	var body []byte
	var err error
	var resp *http.Response

	if req.Body != nil {
		// Reuse request body between retries //
		// attribution: https://stackoverflow.com/a/54706278
		if body, err = ioutil.ReadAll(req.Body); err != nil {
			log.Panic("failed sucking up request body")
		}
		if err = req.Body.Close(); err != nil {
			log.Panic("failed closing request body")
		}
	}

	var lastTime time.Time
	for turn := uint(0); turn < client.Retries; turn++ {
		if req.Body != nil {
			// Recreating request body for each requests //
			req.Body = ioutil.NopCloser(bytes.NewReader(body))
		}

		if turn > 0 {
			sleepDuration := client.BackOff - time.Now().Sub(lastTime)
			if sleepDuration > 0 {
				log.Printf("Retrying %s request in %v %s %s", req.Proto, sleepDuration, req.Method, req.URL.String())
				time.Sleep(time.Duration(sleepDuration))
			}
		}

		// Executes the actual HTTP request here //
		log.Printf("%s request %s %s", req.Proto, req.Method, req.URL.String())
		lastTime = time.Now()
		resp, err = client.HTTPClient.Do(req)

		if err != nil {
			log.Printf("%s request failed: %s", req.Proto, err)
			continue
		}

		if resp == nil {
			log.Panic("http.client.Do() returned nil request and nil error")
		}

		log.Printf("%s response %s (%d bytes) for %s %s",
			resp.Proto, resp.Status, resp.ContentLength, req.Method, req.URL.String())

		if 0 < resp.StatusCode && resp.StatusCode < 300 {
			// success //
			break
		}

		// Some errors shan't be repeated //
		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusMethodNotAllowed,
			http.StatusNotAcceptable, http.StatusProxyAuthRequired, http.StatusGone, http.StatusRequestURITooLong,
			http.StatusTeapot, http.StatusRequestHeaderFieldsTooLarge:
			break
		}

		if e := resp.Body.Close(); e != nil {
			log.Print("Closing of response body failed!")
		}
	}

	if err == nil && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		err = errors.New(fmt.Sprintf("HTTP status code %d received", resp.StatusCode))
	}

	return resp, err
}

func (client *ApiClient) JsonRequest(method string, path string, data interface{}) (
	*http.Response, *json.Decoder, error) {
	var (
		body *bytes.Reader
		err  error
		req  *http.Request
	)

	if data == nil {
		req, err = client.NewRequest(method, path, nil)

	} else {
		// Encode JSON data and present as io.Reader //
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(jsonData)

		req, err = client.NewRequest(method, path, body)
	}
	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, nil, err
	}

	dec, err := decodeJsonResponse(resp)
	if err != nil {
		return resp, nil, err
	}
	return resp, dec, err
}

func decodeJsonResponse(resp *http.Response) (*json.Decoder, error) {
	ctype := resp.Header["Content-Type"]
	if len(ctype) < 1 || strings.ToLower(ctype[0]) == ContentType {
		err := errors.New(fmt.Sprint("Received unknown Content-Type:", ctype))
		log.Print(err)
		return nil, err
	}
	return json.NewDecoder(resp.Body), nil
}
