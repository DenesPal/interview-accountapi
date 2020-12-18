// Copyleft 2020

package interview_accountapi

import "net/url"

// parseURL parses URL string to url.URL struct and the query part to url.Values
func parseURL(urlString string) (*url.URL, url.Values, error) {
	urlStruct, err := url.Parse(urlString)
	if err != nil {
		return nil, nil, err
	}
	q, err := url.ParseQuery(urlStruct.RawQuery)
	if err != nil {
		return nil, nil, err
	}
	return urlStruct, q, nil
}

// assembleURL URL string from url.URL struct and the query part from url.Values
func assembleURL(urlStruct *url.URL, queryValues url.Values) string {
	if queryValues != nil {
		urlStruct.RawQuery = queryValues.Encode()
	}
	return urlStruct.String()
}
