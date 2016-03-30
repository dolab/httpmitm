package httpmitm

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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
	mt.MockRequest("GET", "https://example.com").WithResponse(200, nil, "GET OK")
	mt.MockRequest("GET", "https://example.com/mock").WithResponse(200, nil, "GET MOCK OK")
	mt.MockRequest("PUT", "https://example.com").WithResponse(204, nil, "PUT OK")
	mt.MockRequest("PUT", "https://eXaMpLe.com/mock").WithResponse(200, nil, "PUT MOCK OK")

	// GET /
	response, err := http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("GET OK", string(b))

	// PUT /
	request, _ := http.NewRequest("PUT", "mitm://example.com", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(204, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("PUT OK", string(b))

	// GET /mock
	response, err = http.Get("mitm://example.com/mock")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("GET MOCK OK", string(b))

	// PUT /mock
	request, _ = http.NewRequest("PUT", "mitm://example.cOm/mock", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("PUT MOCK OK", string(b))

	// real http connection
	response, err = http.Head("https://example.com/")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)
	assertion.NotEmpty(response.Header.Get("X-Ec-Custom-Error"))
}

func Test_MitmTransportWithoutResponder(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// GET /
	response, err := http.Get("mitm://example.com")
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)

	// PUT /refuse
	request, _ := http.NewRequest("PUT", "mitm://example.com/refuse", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)
}

func Test_MitmTransportWithTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "http://www.example.com").Times(1).WithResponse(101, nil, "GET OK")

	// GET mitm://www.example.com (mocked)
	response, err := http.Get("mitm://www.example.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)

	// GET mitm://www.example.com (exceeded)
	response, err = http.Get("mitm://www.example.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	// mocks
	mt.MockRequest("PUT", "https://example.com/").WithResponse(101, nil, "PUT OK").Times(2)

	// PUT mitm://example.com (mocked)
	for i := 0; i < 2; i++ {
		request, _ := http.NewRequest("PUT", "mitm://example.com", nil)
		response, err := http.DefaultClient.Do(request)
		assertion.Nil(err)
		assertion.Equal(101, response.StatusCode)
	}

	// PUT mitm://example.com (exceeded)
	request, _ := http.NewRequest("PUT", "mitm://example.com", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(405, response.StatusCode)
}

func Test_MitmTransportWithAnyTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "http://example.com").WithResponse(101, nil, "GET OK").AnyTimes()

	// GET mitm://example.com
	for i := 0; i < 10; i++ {
		response, err := http.Get("mitm://example.com")
		assertion.Nil(err)
		assertion.Equal(101, response.StatusCode)
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
	mt.MockRequest("GET", "http://example.com").WithJsonResponse(200, nil, result{
		Code: 200,
		Name: "OK",
	}).Times(3)

	// GET mitm://example.com
	for i := 0; i < 3; i++ {
		response, err := http.Get("mitm://example.com")
		assertion.Nil(err)
		assertion.Equal(200, response.StatusCode)

		b, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		assertion.Nil(err)
		assertion.Equal(`{"code":200,"name":"OK"}`, string(b))
	}
}

func Test_MitmTransportPauseAndResume(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "https://example.com/").WithResponse(101, nil, "GET OK").AnyTimes()

	// response with mocked
	response, err := http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)

	// paused and response with real github server
	mt.Pause()

	response, err = http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	// resume and response with mock again
	mt.Resume()

	response, err = http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)
}
