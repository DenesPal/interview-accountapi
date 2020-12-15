package interview_accountapi

import (
	"fmt"
	"net/http"
)

func NewApiError(response *http.Response, format string, args ...interface{}) *ApiError {
	var code int
	if response != nil {
		code = response.StatusCode
	}
	return &ApiError{code, fmt.Sprintf(format, args...)}
}

type ApiError struct {
	Code int
	Text string
}

func (err *ApiError) Error() string {
	return err.Text
}
