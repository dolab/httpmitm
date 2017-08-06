package httpmitm

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/url"
	"os"
	"strings"
)

type Testdataer interface {
	io.ReadWriter
}

type Testdata struct {
	r io.Reader
	w io.Writer
}

func NewTestdata(rw io.ReadWriter) *Testdata {
	return &Testdata{
		r: rw.(io.Reader),
		w: rw.(io.Writer),
	}
}

func (td *Testdata) Read(p []byte) (n int, err error) {
	return td.r.Read(p)
}

func (td *Testdata) Write(p []byte) (n int, err error) {
	return td.w.Write(p)
}

func NewTestdataFromIface(v interface{}) (td *Testdata, err error) {
	var reader io.Reader

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

	if err == nil {
		td = &Testdata{
			r: reader,
			w: bytes.NewBufferString(os.DevNull),
		}

		if w, ok := v.(io.Writer); ok {
			td.w = w
		}
	}

	return
}

func NewJsonTestdataFromIface(v interface{}) (td *Testdata, err error) {
	td, err = NewTestdataFromIface(v)
	if err == ErrUnsupport {
		var buf []byte

		buf, err = json.Marshal(v)
		if err == nil {
			td, err = NewTestdataFromIface(buf)
		}
	}

	return
}

func NewXmlTestdataFromIface(v interface{}) (td *Testdata, err error) {
	td, err = NewTestdataFromIface(v)
	if err == ErrUnsupport {
		var buf []byte

		buf, err = xml.Marshal(v)
		if err == nil {
			td, err = NewTestdataFromIface(buf)
		}
	}

	return
}
