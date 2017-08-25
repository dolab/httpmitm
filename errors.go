package httpmitm

import (
	"errors"
)

var (
	ErrNotFound   = errors.New("Not found")
	ErrTimeout    = errors.New("Request timeout")
	ErrUnsupport  = errors.New("Unsupported response data type.")
	ErrRefused    = errors.New("Connection refused. Please making sure the request has been mocked!")
	ErrTimes      = errors.New("Invalid value of times. It must be non-negative integer value.")
	ErrInvocation = errors.New("Not an chained invocation. Please invoking MockRequest(method, url) first.")
)
