// Copyleft 2020

package interview_accountapi

import (
	"fmt"
	"net/http"
)

// NewApiError creates a new ApiError from either http.Response (optional) or error message
func NewApiError(response *http.Response, format string, args ...interface{}) *ApiError {
	var apiErr ApiError

	if response != nil {
		dc, er := decodeJsonResponse(response)
		if er == nil {
			er = dc.Decode(&apiErr)
		}

		apiErr.StatusCode = response.StatusCode
	}

	if apiErr.ErrorMessage == "" {
		if len(args) > 0 {
			apiErr.ErrorMessage = fmt.Sprintf(format, args...)
		} else {
			apiErr.ErrorMessage = format
		}
	}

	return &apiErr
}

// Implements Error interface
func (err *ApiError) Error() string {
	return err.ErrorMessage
}
