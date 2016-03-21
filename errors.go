package httpmitm

import (
	"errors"
)

var (
	ErrUnsupport = errors.New("Unsupported response data type")
	ErrNotFound  = errors.New("No responder found")
	ErrRefused   = errors.New("Connection refused")
	ErrTimeout   = errors.New("Request timeout")
)
