package httpmitm

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/url"
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
	abspath := urlobj.Path
	if abspath == "" {
		abspath = "/"
	}

	key = method + " " + abspath
	return
}

func (td *Testdata) Read(key string) (data []byte, err error) {
	if td.tder != nil {
		return td.tder.Read(key)
	}

	if td.reader != nil {
		data, err = io.ReadAll(td.reader)

		// NOTE: reset reader for internal!
		td.reader = io.NopCloser(bytes.NewBuffer(data))
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

	switch t := v.(type) {
	case Testdataer:
		tder = t

	case io.Reader:
		var data []byte
		data, err = io.ReadAll(t)
		if err == nil {
			reader = bytes.NewBuffer(data)
		}

	case url.Values:
		reader = bytes.NewBuffer([]byte(t.Encode()))

	case string:
		reader = bytes.NewBuffer([]byte(t))

	case []byte:
		reader = bytes.NewBuffer(t)

	default:
		err = ErrUnsupported
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
	if errors.Is(err, ErrUnsupported) {
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
	if errors.Is(err, ErrUnsupported) {
		var buf []byte

		buf, err = xml.Marshal(v)
		if err == nil {
			td, err = NewTestdataFromIface(buf)
		}
	}

	return
}
