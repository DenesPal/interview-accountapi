// Copyleft 2020

package interview_accountapi

import (
	"fmt"
	"net/http"
)

// ApiError serves as an extended error type to save HTTP status code in the Code property
type ApiError struct {
	Code int
	text string
}

// NewApiError creates a new ApiError instance with status code from the http.Response and constructs error message
// from formatted strings
func NewApiError(response *http.Response, format string, args ...interface{}) *ApiError {
	var code int
	if response != nil {
		code = response.StatusCode
	}
	return &ApiError{code, fmt.Sprintf(format, args...)}
}

// Returns error message as string
func (err *ApiError) Error() string {
	return err.text
}
