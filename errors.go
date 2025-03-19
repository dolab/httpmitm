package httpmitm

import (
	"errors"
)

var (
	ErrNotFound    = errors.New("not found. Please making sure the resource has been stubbed")
	ErrTimeout     = errors.New("request timeout")
	ErrUnsupported = errors.New("unsupported Content-Type of response data")
	ErrRefused     = errors.New("connection refused. Please making sure the request has been stubbed")
	ErrTimes       = errors.New("invalid value of times. It must be non-negative integer value")
	ErrInvocation  = errors.New("not an chained invocation. Please invoking MockRequest(method, url) first")
	ErrResponse    = errors.New("not an chained response. Please invoking WithResponser(code, header, body) first")
)
