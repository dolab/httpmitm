package httpmitm

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golib/assert"
)

func Test_NewMitmTransport(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	assertion.Implements((*http.RoundTripper)(nil), mt)
}

func Test_MitmTransportStubDefaultTransport(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	defer mt.UnstubDefaultTransport()

	mt.StubDefaultTransport(t)
	assertion.Equal(mt, http.DefaultTransport)
}

func Test_MitmTransport(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", mockURL).WithResponse(200, nil, "GET OK")
	mt.MockRequest("GET", mockURL+"/mock").WithResponse(200, nil, "GET MOCK OK")
	mt.MockRequest("PUT", mockURL).WithResponse(204, nil, "PUT OK")
	mt.MockRequest("PUT", mockURL+"/mock").WithResponse(200, nil, "PUT MOCK OK")

	// GET /
	response, err := http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("GET OK", string(b))

	// PUT /
	request, _ := http.NewRequest("PUT", stubURL, nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(204, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("PUT OK", string(b))

	// GET /mock
	response, err = http.Get(stubURL + "/mock")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("GET MOCK OK", string(b))

	// PUT /mock
	request, _ = http.NewRequest("PUT", stubURL+"/mock", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("PUT MOCK OK", string(b))

	// real http connection
	response, err = http.Head(stubURL + "/httpmitm")
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)
}

func Test_MitmTransportWithMultiple(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// first, it should work
	mt.MockRequest("GET", mockURL).WithResponse(200, nil, "GET OK")

	response, err := http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	// seconde, it should response with 404
	mt.MockRequest("GET", mockURL).WithResponse(404, nil, "Not Found")

	response, err = http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(404, response.StatusCode)
}

func Test_MitmTransportWithoutResponder(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// GET /
	response, err := http.Get(stubURL)
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)

	// PUT /refuse
	request, _ := http.NewRequest("PUT", stubURL+"/refuse", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)
}

func Test_MitmTransportWithTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", mockURL).Times(1).WithResponse(101, nil, "MOCK OK")

	// GET mitm://example.com (mocked)
	response, err := http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)
	assertion.ReaderContains(response.Body, "MOCK OK")
	response.Body.Close()

	// // GET mitm://example.com (exceeded)
	// response, err = http.Get(stubURL)
	// assertion.Nil(err)
	// assertion.Equal(200, response.StatusCode)
	// assertion.ReaderContains(response.Body, "GET OK")
	// response.Body.Close()

	// mocks
	mt.MockRequest("PUT", mockURL).WithResponse(101, nil, "MOCK OK").Times(2)

	// PUT mitm://example.com (mocked)
	for i := 0; i < 2; i++ {
		request, _ := http.NewRequest("PUT", stubURL, nil)

		response, err := http.DefaultClient.Do(request)
		assertion.Nil(err)
		assertion.Equal(101, response.StatusCode)
		assertion.ReaderContains(response.Body, "MOCK OK")
		response.Body.Close()
	}

	// // PUT mitm://example.com (exceeded)
	// request, _ := http.NewRequest("PUT", stubURL, nil)
	// response, err = http.DefaultClient.Do(request)
	// assertion.Nil(err)
	// assertion.Equal(200, response.StatusCode)
	// assertion.ReaderContains(response.Body, "PUT OK")
}

func Test_MitmTransportWithAnyTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", mockURL).WithResponse(101, nil, "MOCK OK").AnyTimes()

	// GET mitm://example.com
	for i := 0; i < 10; i++ {
		response, err := http.Get(stubURL)
		assertion.Nil(err)
		assertion.Equal(101, response.StatusCode)
		assertion.ReaderContains(response.Body, "MOCK OK")
		response.Body.Close()

	}
}

func Test_MitmTransportWithTimesError(t *testing.T) {
	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	assertion := assert.New(t)

	type result struct {
		Code int    `json:"code"`
		Name string `json:"name"`
	}

	// mocks
	mt.MockRequest("GET", mockURL).WithJsonResponse(200, nil, result{
		Code: 200,
		Name: "OK",
	}).Times(3)

	// GET mitm://example.com
	for i := 0; i < 3; i++ {
		response, err := http.Get(stubURL)
		assertion.Nil(err)
		assertion.Equal(200, response.StatusCode)
		assertion.ReaderContains(response.Body, `{"code":200,"name":"OK"}`)
	}
}

func Test_MitmTransportPauseAndResume(t *testing.T) {
	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	assertion := assert.New(t)

	// mocks
	mt.MockRequest("GET", mockURL).WithResponse(101, nil, "MOCK OK").AnyTimes()

	// response with mocked
	response, err := http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)
	assertion.ReaderContains(response.Body, "MOCK OK")
	response.Body.Close()

	// paused and response with real github server
	mt.Pause()

	response, err = http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)
	assertion.ReaderContains(response.Body, "GET OK")
	response.Body.Close()

	// resume and response with mock again
	mt.Resume()

	response, err = http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)
	assertion.ReaderContains(response.Body, "MOCK OK")
	response.Body.Close()
}

func Test_MitmTransportWithTestdataer(t *testing.T) {
	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	assertion := assert.New(t)

	// mocks
	mt.MockRequest("GET", mockURL).WithResponse(200, nil, apidata)

	// response with mocked
	response, err := http.Get(stubURL)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)
	assertion.ReaderContains(response.Body, "Hello, httpmitm!")
}
