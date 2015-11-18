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

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "mitm://example.com").WithResponse(200, nil, "GET OK")
	mt.MockRequest("PUT", "http://example.com").WithResponse(204, nil, "PUT OK")
	mt.MockRequest("GET", "http://example.com/mock").WithResponse(200, nil, "GET MOCK OK")
	mt.MockRequest("PUT", "http://eXaMpLe.com/mock").WithResponse(200, nil, "PUT MOCK OK")

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
	response, err = http.Head("https://github.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)
	assertion.NotEmpty(response.Header.Get("X-Github-Request-Id"))
}

func Test_MitmTransportWithoutResponder(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// GET /refuse
	response, err := http.Get("mitm://example.com/refuse")
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)

	// PUT /refuse
	request, _ := http.NewRequest("PUT", "mitm://example.com/refuse", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Contains(err.Error(), ErrRefused.Error())
	assertion.Nil(response)
}

func Test_MitmTransportWithDefaultResponder(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	mt.SetDefaultResponder(NewResponder(100, nil, "DEFAULT OK"))

	// GET /unmocked
	response, err := http.Get("mitm://example.com/unmocked")
	assertion.Nil(err)
	assertion.Equal(100, response.StatusCode)

	b, err := ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("DEFAULT OK", string(b))

	// PUT /unmocked
	request, _ := http.NewRequest("PUT", "mitm://example.com/unmocked", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(100, response.StatusCode)

	b, err = ioutil.ReadAll(response.Body)
	response.Body.Close()

	assertion.Nil(err)
	assertion.Equal("DEFAULT OK", string(b))
}

func Test_MitmTransportWithTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "http://example.com").Times(1).WithResponse(101, nil, "GET OK")
	mt.MockRequest("PUT", "https://github.com").WithResponse(101, nil, "PUT OK").Times(2)

	// GET mitm://example.com (mocked)
	response, err := http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)

	// GET mitm://example.com (exceeded)
	response, err = http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	// PUT mitm://github.com (mocked)
	for i := 0; i < 2; i++ {
		request, _ := http.NewRequest("PUT", "mitm://github.com", nil)
		response, err := http.DefaultClient.Do(request)
		assertion.Nil(err)
		assertion.Equal(101, response.StatusCode)
	}

	// PUT mitm://github.com (exceeded)
	request, _ := http.NewRequest("PUT", "mitm://github.com", nil)
	response, err = http.DefaultClient.Do(request)
	assertion.Nil(err)
	assertion.Equal(404, response.StatusCode)
}

func Test_MitmTransportWithAnyTimes(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
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

func Test_MitmTransportMissMatched(t *testing.T) {
	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// this should fail test case
	// mt.MockRequest("GET", "http://example.com").WithResponse(101, nil, "GET OK").Times(1)
	// mt.MockRequest("PUT", "http://example.com").WithResponse(101, nil, "GET OK").Times(1)
}

func Test_MitmTransportPauseAndResume(t *testing.T) {
	assertion := assert.New(t)

	mt := NewMitmTransport()
	mt.StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	// mocks
	mt.MockRequest("GET", "https://github.com").WithResponse(101, nil, "GET OK").AnyTimes()

	// response with mocked
	response, err := http.Get("mitm://github.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)

	// paused and response with real github server
	mt.Pause()

	response, err = http.Get("mitm://github.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)

	// resume and response with mock again
	mt.Resume()

	response, err = http.Get("mitm://github.com")
	assertion.Nil(err)
	assertion.Equal(101, response.StatusCode)
}
