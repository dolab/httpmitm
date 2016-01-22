package httpmitm

import (
	"net/http"
	"sync"
)

// Responser is an interface representing the ability to
// execute a single HTTP transaction, mocking the Response for a given Request.
type Responser interface {
	http.RoundTripper
}

type Response struct {
	mux sync.Mutex

	responder     Responser
	scheme        string // origin url scheme
	expectedTimes int    // expected mock times
	invokedTimes  int    // really mocked times
}

func NewResponse(response Responser, scheme string, times int) *Response {
	return &Response{
		responder:     response,
		scheme:        scheme,
		expectedTimes: times,
		invokedTimes:  0,
	}
}

func (res *Response) Scheme() string {
	return res.scheme
}

func (res *Response) MatchTimes() bool {
	return res.expectedTimes == MockUnlimitedTimes || res.expectedTimes == res.invokedTimes
}

func (res *Response) Times() (expected, invoked int) {
	return res.expectedTimes, res.invokedTimes
}

func (res *Response) SetExpectedTimes(expected int) {
	res.expectedTimes = expected
}

func (res *Response) RoundTrip(r *http.Request) (*http.Response, error) {
	res.mux.Lock()
	defer res.mux.Unlock()

	// unlimited mock
	if res.expectedTimes == MockUnlimitedTimes {
		res.invokedTimes += 1

		return res.responder.RoundTrip(r)
	}

	// direct connect when response mock times is reached
	if res.invokedTimes == res.expectedTimes {
		r.URL.Scheme = res.scheme

		return httpDefaultResponder.RoundTrip(r)
	}

	res.invokedTimes += 1

	return res.responder.RoundTrip(r)
}
