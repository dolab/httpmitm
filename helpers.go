package httpmitm

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

func (_ *_Helper) NewJsonReaderFromIface(v interface{}) (reader io.Reader, err error) {
	switch v.(type) {
	case string:
		s, _ := v.(string)

		reader = strings.NewReader(s)

	case []byte:
		b, _ := v.([]byte)

		reader = bytes.NewReader(b)

	case url.Values:
		params, _ := v.(url.Values)

		var buf []byte

		buf, err = json.Marshal(params)
		reader = bytes.NewReader(buf)

	case io.Reader:
		reader, _ = v.(io.Reader)

	default:
		var buf []byte

		buf, err = json.Marshal(v)
		reader = bytes.NewReader(buf)

	}

	return
}

func (_ *_Helper) NewXmlReaderFromIface(v interface{}) (reader io.Reader, err error) {
	switch v.(type) {
	case string:
		s, _ := v.(string)

		reader = strings.NewReader(s)

	case []byte:
		b, _ := v.([]byte)

		reader = bytes.NewReader(b)

	case url.Values:
		params, _ := v.(url.Values)

		var buf []byte

		buf, err = xml.Marshal(params)
		reader = bytes.NewReader(buf)

	case io.Reader:
		reader, _ = v.(io.Reader)

	default:
		var buf []byte

		buf, err = xml.Marshal(v)
		reader = bytes.NewReader(buf)

	}

	return
}
