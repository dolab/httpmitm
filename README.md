# httpmitm

[![CircleCI](https://circleci.com/gh/dolab/httpmitm.svg?style=svg)](https://circleci.com/gh/dolab/httpmitm)

HTTP mock framework for golang.

## Assuming
- httpmitm assume your requested *URL* scheme is `mitm` (**M**an-**I**n-**T**he-**M**iddle). Thus, you must make a request to `mitm://github.com` when you want to mock a request of `https://github.com`.
- httpmitm treats request *URL* in case-insensitive

## Usage
```go
import (
    "testing"

    "github.com/dolab/httpmitm"
    "github.com/golib/assert"
)

func Test_MitmTransport(t *testing.T) {
    // stub http.DefaultTransport as early as possible
    mt := httpmitm.NewMitmTransport().StubDefaultTransport(t)
    defer mt.UnstubDefaultTransport()

    // mocks, your can use http, https and mitm scheme here
    mt.MockRequest("GET", "mitm://example.com").WithResponse(200, nil, "GET OK")
    mt.MockRequest("PUT", "mitm://example.com").WithResponse(204, nil, "PUT OK")
    mt.MockRequest("GET", "http://example.com/mock").WithResponse(200, nil, "GET MOCK OK")
    mt.MockRequest("PUT", "http://eXaMpLe.com/mock").WithResponse(200, nil, "PUT MOCK OK")

    assertion := assert.New(t)

    // GET mitm://example.com
    response, err := http.Get("mitm://example.com")
    assertion.Nil(err)
    assertion.Equal(200, response.StatusCode)

    b, err := ioutil.ReadAll(response.Body)
    response.Body.Close()

    assertion.Nil(err)
    assertion.Equal("GET OK", string(b))

    // PUT mitm://example.com
    request, _ := http.NewRequest("PUT", "mitm://example.com", nil)
    response, err = http.DefaultClient.Do(request)
    assertion.Nil(err)
    assertion.Equal(204, response.StatusCode)

    b, err = ioutil.ReadAll(response.Body)
    response.Body.Close()

    assertion.Nil(err)
    assertion.Equal("PUT OK", string(b))

    // GET mitm://example.com/mock
    response, err = http.Get("miTm://example.com/mock")
    assertion.Nil(err)
    assertion.Equal(200, response.StatusCode)

    b, err = ioutil.ReadAll(response.Body)
    response.Body.Close()

    assertion.Nil(err)
    assertion.Equal("GET MOCK OK", string(b))

    // PUT mitm://example.cOm/mock
    request, _ = http.NewRequest("PUT", "mitm://example.cOm/mock", nil)
    response, err = http.DefaultClient.Do(request)
    assertion.Nil(err)
    assertion.Equal(200, response.StatusCode)

    b, err = ioutil.ReadAll(response.Body)
    response.Body.Close()

    assertion.Nil(err)
    assertion.Equal("PUT MOCK OK", string(b))

    // no mock, real http connection
    response, err = http.Head("https://github.com")
    assertion.Nil(err)
    assertion.Equal(200, response.StatusCode)
    assertion.NotEmpty(response.Header.Get("X-Github-Request-Id"))
}

func Test_MitmTransportWithTimes(t *testing.T) {
    // stub http.DefaultTransport as early as possible
    mt := NewMitmTransport().StubDefaultTransport(t)
    defer mt.UnstubDefaultTransport()

    // mock once
    mt.MockRequest("GET", "http://example.com").Times(1).WithResponse(101, nil, "GET OK")
    // mock 2 times
    mt.MockRequest("PUT", "https://github.com").WithResponse(101, nil, "PUT OK").Times(2)
    // mock forever
    mt.MockRequest("GET", "https://google.com").WithResponse(101, nil, "PUT OK").AnyTimes()

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

    // GET mitm://google.com (forever mocked)
    for i := 0; i < 10; i++ {
        response, err = http.Get("mitm://google.com")
        assertion.Nil(err)
        assertion.Equal(101, response.StatusCode)
    }
}

func Test_MitmTransportPauseAndResume(t *testing.T) {
    mt := NewMitmTransport().StubDefaultTransport(t)
    defer mt.UnstubDefaultTransport()

    assertion := assert.New(t)

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
```

## Using `Testdataer`

*httpmitm* supports custom response by implementing a `Testdataer` interface.

- implements `Testdataer`

```go
// ApiData implements Testdataer
type ApiData struct {
    contents map[string][]byte
}

func NewApiData(data map[string][]byte) *ApiData {
    return &ApiData{
        contents: data,
    }
}

func (data *ApiData) Key(method string, urlobj *url.URL) (key string) {
    abspath := urlobj.Path
    if abspath == "" {
        abspath = "/"
    }

    return fmt.Sprintf("%s %s", method, abspath)
}

func (data *ApiData) Read(key string) (data []byte, err error) {
    data, ok := data.contents[key]
    if !ok {
        err = errors.New("Not found")
    }

    return
}

func (data *ApiData) Write(key string, data []byte) (err error) {
    data.contents[key] = data

    return nil
}
```

- using `ApiData`

```go
import (
    "testing"

    "github.com/dolab/httpmitm"
    "github.com/golib/assert"
)

// usage
var (
    api = NewApiData(map[string][]byte{
        "GET /": []byte{"Hello, httpmitm!"},
    })
)

func Test_MitmTransportWithTestdataer(t *testing.T) {
	mt := httpmitm.NewMitmTransport().StubDefaultTransport(t)
	defer mt.UnstubDefaultTransport()

	assertion := assert.New(t)

	// mocks
	mt.MockRequest("GET", "http://example.com").WithResponse(200, nil, apidata)

	// response with mocked
	response, err := http.Get("mitm://example.com")
	assertion.Nil(err)
	assertion.Equal(200, response.StatusCode)
	assertion.ReaderContains(response.Body, "Hello, httpmitm!")
}
```

## TODO

- [ ] support wildcard pattern with resource url
- [x] support callback response type
- [ ] support named params for callback response
- [x] support custom response data driver

## Author
[Spring MC](https://twitter.com/mcspring)
