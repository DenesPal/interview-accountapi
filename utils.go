package interview_accountapi

import "net/url"

// Parses sting URLs to URL struct and query values //
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

func assembleURL(urlStruct *url.URL, queryValues url.Values) string {
	if queryValues != nil {
		urlStruct.RawQuery = queryValues.Encode()
	}
	return urlStruct.String()
}
