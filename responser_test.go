package httpmitm

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/golib/assert"
)

type testResponserRounderTrip struct{}

func (trrt *testResponserRounderTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, nil
}

func Test_NewResponser(t *testing.T) {
	it := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := mockURL
	times := 1

	responser := NewResponser(responder, rawurl, times)
	it.Implements((*http.RoundTripper)(nil), responser)

	mocker := responser.mocks["/"]
	it.Equal(rawurl, mocker.rawurl)
	it.NotNil(mocker.matcher)
	it.Equal("http", mocker.originScheme)
	it.Equal(times, mocker.expectedTimes)
	it.Equal(0, mocker.invokedTimes)
}

func Test_ResponserNew(t *testing.T) {
	it := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := mockURL
	times := 1

	responser := NewResponser(responder, rawurl, times)
	it.Implements((*http.RoundTripper)(nil), responser)
	it.Equal(1, len(responser.mocks))

	responser.New(responder, mockURL+"/newpath", times)
	it.Equal(2, len(responser.mocks))
	it.NotNil(responser.mocks["/newpath"])
}

func Test_ResponserSetMatcherByRawURL(t *testing.T) {
	it := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := mockURL
	times := 1

	var matcher RequestMatcher = func(r *http.Request, urlobj *url.URL) bool {
		return true
	}

	responser := NewResponser(responder, rawurl, times)
	responser.New(responder, mockURL+"/newpath", 1)

	responser.SetMatcherByRawURL(rawurl, matcher)
	it.Condition(func() bool {
		return fmt.Sprintf("%p", matcher) == fmt.Sprintf("%p", responser.mocks["/"].matcher)
	})
	it.Condition(func() bool {
		return fmt.Sprintf("%p", matcher) != fmt.Sprintf("%p", responser.mocks["/newpath"].matcher)
	})
}

func Test_ResponserSetExpectedTimesByRawURL(t *testing.T) {
	it := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := mockURL
	times := 1

	responser := NewResponser(responder, rawurl, times)
	responser.New(responder, mockURL+"/newpath", 1)

	responser.SetExpectedTimesByRawURL(rawurl, 2)
	it.Equal(2, responser.mocks["/"].expectedTimes)
	it.Equal(1, responser.mocks["/newpath"].expectedTimes)
}

func Test_ResponserFind(t *testing.T) {
	it := assert.New(t)
	responder := new(testResponserRounderTrip)
	responser := NewResponser(responder, mockURL, 1)
	responser.New(responder, mockURL+"/newpath", 1)

	// exist path
	mocker := responser.Find("/newpath")
	it.NotNil(mocker)

	// unexist path, returns default mocker if exists
	mocker = responser.Find("/htapwen")
	it.Equal(responser.Find("/"), mocker)
}

func Test_RefusedResponser(t *testing.T) {
	it := assert.New(t)

	it.Implements((*http.RoundTripper)(nil), RefusedResponser)

	request, _ := http.NewRequest("GET", mockURL, nil)
	response, err := RefusedResponser.RoundTrip(request)
	it.EqualError(err, ErrRefused.Error())
	it.Nil(response)
}

func Test_TimeoutResponser(t *testing.T) {
	it := assert.New(t)

	it.Implements((*http.RoundTripper)(nil), RefusedResponser)

	request, _ := http.NewRequest("GET", mockURL, nil)
	response, err := TimeoutResponser.RoundTrip(request)
	it.EqualError(err, ErrTimeout.Error())
	it.Nil(response)
}
