package httpmitm

import (
	"errors"
)

var (
	ErrNotFound   = errors.New("Not found. Please making sure the resource has been stubed!")
	ErrTimeout    = errors.New("Request timeout")
	ErrUnsupport  = errors.New("Unsupported Conten-Type of response data.")
	ErrRefused    = errors.New("Connection refused. Please making sure the request has been stubed!")
	ErrTimes      = errors.New("Invalid value of times. It must be non-negative integer value.")
	ErrInvocation = errors.New("Not an chained invocation. Please invoking MockRequest(method, url) first.")
	ErrResponse   = errors.New("Not an chained response. Please invoking WithResponser(code, header, body) first.")
)
