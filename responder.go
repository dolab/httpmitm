package httpmitm

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

var (
	httpDefaultResponder = http.DefaultTransport // internal
)

// Responder defines response of mocked request
// NOTE: Responder implements http.RoundTripper for invokation chainning.
type Responder struct {
	code   int
	header http.Header
	body   Testdataer
	err    error
	callee func(r *http.Request) (code int, header http.Header, reader io.Reader, err error)
}

// NewResponder returns Responder with provided data
func NewResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	tder, err := NewTestdataFromIface(body)

	if header == nil {
		header = http.Header{}
	}

	return &Responder{
		code:   code,
		header: header,
		body:   tder,
		err:    err,
	}
}

// NewJsonResponder returns Responder with json.Marshal(body) format
func NewJsonResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	tder, err := NewJsonTestdataFromIface(body)

	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "application/json")

	return &Responder{
		code:   code,
		header: header,
		body:   tder,
		err:    err,
	}
}

// NewXmlResponder returns Responder with xml.Marshal(body) format
func NewXmlResponder(code int, header http.Header, body interface{}) http.RoundTripper {
	tder, err := NewXmlTestdataFromIface(body)

	if header == nil {
		header = http.Header{}
	}

	// overwrite response content type
	header.Set("Content-Type", "text/xml")

	return &Responder{
		code:   code,
		header: header,
		body:   tder,
		err:    err,
	}
}

// NewCalleeResponder returns Responder with callee which invoked with mocked request
func NewCalleeResponder(callee func(r *http.Request) (code int, header http.Header, body io.Reader, err error)) http.RoundTripper {
	return &Responder{
		callee: callee,
		body:   NewTestdata(bytes.NewReader([]byte{})),
	}
}

func (r *Responder) Write(method string, urlobj *url.URL, data []byte) error {
	key := r.body.Key(method, urlobj)

	return r.body.Write(key, data)
}

func (r *Responder) RoundTrip(req *http.Request) (*http.Response, error) {
	// is there an error?
	if r.err != nil {
		return nil, r.err
	}

	key := r.body.Key(req.Method, req.URL)

	// apply callee if exists
	if r.callee != nil {
		var (
			reader io.Reader
		)

		r.code, r.header, reader, r.err = r.callee(req)
		if r.err != nil {
			return nil, r.err
		}

		// sync reader data returned by callee to r.body
		td, ok := r.body.(*Testdata)
		if ok {
			td.reader = reader
		}
	}

	data, err := r.body.Read(key)
	if err != nil {
		return nil, err
	}

	// push back for response reader
	response := &http.Response{
		Status:     strconv.Itoa(r.code),
		StatusCode: r.code,
		Header:     r.header,
		Body:       ioutil.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}

	// adjust response content length header if missed
	if _, ok := response.Header["Content-Length"]; !ok {
		response.Header.Add("Content-Length", strconv.Itoa(len(data)))
	}
	response.ContentLength, _ = strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64)

	return response, nil
}

// NotFoundResponder represents a connection with 404 reponse.
type NotFoundResponder struct{}

func NewNotFoundResponder() *NotFoundResponder {
	return &NotFoundResponder{}
}

func (nf *NotFoundResponder) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, ErrNotFound
}

// RefusedResponder represents a connection failure response of mocked request.
// NOTE: It's used as default Responder for empty mock.
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
