package httpmitm

import (
	"errors"
)

var (
	ErrUnsupport  = errors.New("Unsupported response data type")
	ErrNotFound   = errors.New("No responder found")
	ErrRefused    = errors.New("Connection refused")
	ErrTimeout    = errors.New("Request timeout")
	ErrTimes      = errors.New("Invalid value of times. It must be non-negative integer value.")
	ErrInvocation = errors.New("Not an chained invocation. Please invoking MockRequest(method, url) first.")
)
