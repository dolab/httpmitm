package httpmitm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testResponserRounderTrip struct{}

func (trrt *testResponserRounderTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, nil
}

func Test_NewResponser(t *testing.T) {
	assertion := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := "https://example.com"
	times := 1

	responser := NewResponser(responder, rawurl, times)
	assertion.Implements((*http.RoundTripper)(nil), responser)

	mocker := responser.mocks["/"]
	assertion.Equal(rawurl, mocker.rawurl)
	assertion.NotNil(mocker.matcher)
	assertion.Equal("https", mocker.originScheme)
	assertion.Equal(times, mocker.expectedTimes)
	assertion.Equal(0, mocker.invokedTimes)
}

func Test_ResponserNew(t *testing.T) {
	assertion := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := "https://example.com"
	times := 1

	responser := NewResponser(responder, rawurl, times)
	assertion.Implements((*http.RoundTripper)(nil), responser)
	assertion.Equal(1, len(responser.mocks))

	responser.New(responder, "https://example.com/newpath", times)
	assertion.Equal(2, len(responser.mocks))
	assertion.NotNil(responser.mocks["/newpath"])
}

func Test_ResponserSetExpectedTimesByRawURL(t *testing.T) {
	assertion := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := "https://example.com"
	times := 1

	responser := NewResponser(responder, rawurl, times)
	responser.New(responder, "https://example.com/newpath", 1)

	responser.SetExpectedTimesByRawURL(rawurl, 2)
	assertion.Equal(2, responser.mocks["/"].expectedTimes)
	assertion.Equal(1, responser.mocks["/newpath"].expectedTimes)
}

func Test_ResponserSetRequestMatcherByRawURL(t *testing.T) {
	assertion := assert.New(t)
	responder := new(testResponserRounderTrip)
	rawurl := "https://example.com"
	times := 1

	var matcher RequestMatcher = func(r *http.Request, rawurl string) bool {
		return true
	}

	responser := NewResponser(responder, rawurl, times)
	responser.New(responder, "https://example.com/newpath", 1)

	responser.SetRequestMatcherByRawURL(rawurl, matcher)
	assertion.Condition(func() bool {
		return fmt.Sprintf("%p", matcher) == fmt.Sprintf("%p", responser.mocks["/"].matcher)
	})
	assertion.Condition(func() bool {
		return fmt.Sprintf("%p", matcher) != fmt.Sprintf("%p", responser.mocks["/newpath"].matcher)
	})
}

func Test_ResponserFind(t *testing.T) {
	assertion := assert.New(t)
	responder := new(testResponserRounderTrip)
	responser := NewResponser(responder, "https://example.com", 1)
	responser.New(responder, "https://example.com/newpath", 1)

	// exist path
	mocker := responser.Find("/newpath")
	assertion.NotNil(mocker)

	// unexist path, returns default mocker if exists
	mocker = responser.Find("/htapwen")
	assertion.Equal(responser.Find("/"), mocker)
}

func Test_RefusedResponser(t *testing.T) {
	assertion := assert.New(t)

	assertion.Implements((*http.RoundTripper)(nil), RefusedResponser)

	request, _ := http.NewRequest("GET", "https://example.com", nil)
	response, err := RefusedResponser.RoundTrip(request)
	assertion.EqualError(err, ErrRefused.Error())
	assertion.Nil(response)
}
