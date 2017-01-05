package httpmitm

import (
	"bytes"
	"io"
	"net/url"
	"strings"
)

var (
	Helpers *_Helper
)

type _Helper struct{}

func (_ *_Helper) NewReaderFromIface(v interface{}) (reader io.Reader, err error) {
	switch v.(type) {
	case string:
		s, _ := v.(string)

		reader = strings.NewReader(s)

	case []byte:
		b, _ := v.([]byte)

		reader = bytes.NewReader(b)

	case url.Values:
		params, _ := v.(url.Values)

		reader = strings.NewReader(params.Encode())

	case io.Reader:
		reader, _ = v.(io.Reader)

	default:
		err = ErrUnsupport

	}

	return
}
