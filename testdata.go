package httpmitm

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
)

type Testdataer interface {
	Key(method string, urlobj *url.URL) (key string)
	Read(key string) (data []byte, err error)
	Write(key string, data []byte) (err error)
}

type Testdata struct {
	tder   Testdataer
	reader io.Reader
}

func NewTestdata(r io.Reader) *Testdata {
	return &Testdata{
		reader: r,
	}
}

func (td *Testdata) Key(method string, urlobj *url.URL) (key string) {
	if td.tder != nil {
		return td.tder.Key(method, urlobj)
	}

	// default to "METHOD /path/to/resource"
	key = method + " " + urlobj.Path
	return
}

func (td *Testdata) Read(key string) (data []byte, err error) {
	if td.tder != nil {
		return td.tder.Read(key)
	}

	if td.reader != nil {
		data, err = ioutil.ReadAll(td.reader)

		// NOTE: reset reader for internal!
		td.reader = ioutil.NopCloser(bytes.NewReader(data))
	} else {
		err = io.EOF
	}

	return
}

func (td *Testdata) Write(key string, data []byte) (err error) {
	if td.tder != nil {
		return td.tder.Write(key, data)
	}

	// NOTE: ignore for internal

	return
}

func NewTestdataFromIface(v interface{}) (td *Testdata, err error) {
	var (
		tder   Testdataer
		reader io.Reader
	)

	switch v.(type) {
	case Testdataer:
		tder, _ = v.(Testdataer)

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

	if err != nil {
		return
	}

	td = &Testdata{
		tder:   tder,
		reader: reader,
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
