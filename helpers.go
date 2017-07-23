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
	case io.Reader:
		reader, _ = v.(io.Reader)

	case url.Values:
		params, _ := v.(url.Values)

		reader = strings.NewReader(params.Encode())

	case string:
		s, _ := v.(string)

		reader = strings.NewReader(s)

	case []byte:
		b, _ := v.([]byte)

		reader = bytes.NewReader(b)

	default:
		err = ErrUnsupport

	}

	return
}

func (_ *_Helper) NewJsonReaderFromIface(v interface{}) (reader io.Reader, err error) {
	reader, err = Helpers.NewReaderFromIface(v)
	if err == ErrUnsupport {
		var buf []byte

		buf, err = json.Marshal(v)
		if err == nil {
			reader = bytes.NewReader(buf)
		}
	}

	return
}

func (_ *_Helper) NewXmlReaderFromIface(v interface{}) (reader io.Reader, err error) {
	reader, err = Helpers.NewReaderFromIface(v)
	if err == ErrUnsupport {
		var buf []byte

		buf, err = xml.Marshal(v)
		if err == nil {
			reader = bytes.NewReader(buf)
		}
	}

	return
}
