package httpmitm

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

var (
	RefuseResponse = NewResponse(NewRefuseResponder(), MockScheme, MockUnlimitedTimes)

	httpDefaultResponder = http.DefaultTransport // internal
)

type Responder struct {
	code   int
	header http.Header
	body   []byte
	err    error
}

func NewResponder(code int, header http.Header, body interface{}) Responser {
	var (
		buf []byte
		err error
	)

	switch body.(type) {
	case io.Reader:
		reader, _ := body.(io.Reader)

		buf, err = ioutil.ReadAll(reader)

	case string:
		s, _ := body.(string)

		buf = []byte(s)

	case []byte:
		buf, _ = body.([]byte)

	case url.Values:
		params, _ := body.(url.Values)

		buf = []byte(params.Encode())

	default:
		err = ErrUnsupport

	}

	if header == nil {
		header = http.Header{}
	}

	if _, ok := header["Content-Type"]; !ok {
		header.Set("Content-Type", "text/plain")
	}

	return &Responder{
		code:   code,
		header: header,
		body:   buf,
		err:    err,
	}
}

func NewJsonResponder(code int, header http.Header, body interface{}) Responser {
	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "application/json")

	buf, err := json.Marshal(body)

	return &Responder{
		code:   code,
		header: header,
		body:   buf,
		err:    err,
	}
}

func NewXmlResponder(code int, header http.Header, body interface{}) Responser {
	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "application/xml")

	buf, err := xml.Marshal(body)

	return &Responder{
		code:   code,
		header: header,
		body:   buf,
		err:    err,
	}
}

func (rr *Responder) RoundTrip(r *http.Request) (*http.Response, error) {
	if rr.err != nil {
		return nil, rr.err
	}

	response := &http.Response{
		Status:     strconv.Itoa(rr.code),
		StatusCode: rr.code,
		Header:     rr.header,
		Body:       ioutil.NopCloser(bytes.NewBuffer(rr.body)),
	}

	// adjust response content length
	contentLength := len(rr.body)
	response.ContentLength = int64(contentLength)
	response.Header.Set("Content-Length", strconv.Itoa(contentLength))

	return response, nil
}

// RefuseResponder represents a connection failure of request and response
type RefuseResponder struct {
}

func NewRefuseResponder() *RefuseResponder {
	return &RefuseResponder{}
}

func (rr *RefuseResponder) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, ErrRefused
}
