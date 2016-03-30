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
	"strings"
)

var (
	httpDefaultResponder = http.DefaultTransport // internal
)

// Responder defines response of mocked request
// NOTE: Responder implements http.RoundTripper for invokation chainning.
type Responder struct {
	code   int
	header http.Header
	body   io.Reader
	err    error
	callee func(r *http.Request) (code int, header http.Header, reader io.Reader, err error)
}

// NewResponder returns Responder with provided data
func NewResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	var (
		reader io.Reader
		err    error
	)

	switch body.(type) {
	case string:
		s, _ := body.(string)

		reader = strings.NewReader(s)

	case []byte:
		b, _ := body.([]byte)

		reader = bytes.NewReader(b)

	case url.Values:
		params, _ := body.(url.Values)

		reader = strings.NewReader(params.Encode())

	case io.Reader:
		reader, _ = body.(io.Reader)

	default:
		err = ErrUnsupport

	}

	if header == nil {
		header = http.Header{}
	}

	return &Responder{
		code:   code,
		header: header,
		body:   reader,
		err:    err,
	}
}

// NewJsonResponder returns Responder with json.Marshal(body) format
func NewJsonResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "application/json")

	b, err := json.Marshal(body)

	return &Responder{
		code:   code,
		header: header,
		body:   bytes.NewReader(b),
		err:    err,
	}
}

// NewXmlResponder returns Responder with xml.Marshal(body) format
func NewXmlResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "text/xml")

	b, err := xml.Marshal(body)

	return &Responder{
		code:   code,
		header: header,
		body:   bytes.NewReader(b),
		err:    err,
	}
}

// NewCalleeResponder returns Responder with callee which invoked with mocked request
func NewCalleeResponder(callee func(r *http.Request) (code int, header http.Header, body io.Reader, err error)) http.RoundTripper {
	return &Responder{
		callee: callee,
	}
}

func (r *Responder) RoundTrip(req *http.Request) (*http.Response, error) {
	// apply callee if exists
	if r.callee != nil {
		r.code, r.header, r.body, r.err = r.callee(req)
	}

	if r.err != nil {
		return nil, r.err
	}

	response := &http.Response{
		Status:     strconv.Itoa(r.code),
		StatusCode: r.code,
		Header:     r.header,
	}

	if body, ok := r.body.(io.ReadCloser); ok {
		response.Body = body
	} else {
		response.Body = ioutil.NopCloser(r.body)
	}

	// adjust response content length header if missed
	if _, ok := response.Header["Content-Length"]; !ok {
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		response.Body.Close()

		response.Header.Add("Content-Length", strconv.Itoa(len(b)))
		response.ContentLength = int64(len(b))

		// pull back for response reader
		response.Body = ioutil.NopCloser(bytes.NewReader(b))
	}

	return response, nil
}

// RefusedResponder represents a connection failure response of mocked request.
// It's used as default Responder for empty mock.
type RefusedResponder struct{}

func NewRefusedResponder() *RefusedResponder {
	return &RefusedResponder{}
}

func (r *RefusedResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, ErrRefused
}

// TimeoutResponder represents a connection timeout response of mocked request.
type TimeoutResponder struct{}

func NewTimeoutResponder() *TimeoutResponder {
	return &TimeoutResponder{}
}

func (t *TimeoutResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, ErrTimeout
}
