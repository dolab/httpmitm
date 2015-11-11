package httpmitm

import (
	"net/http"
	"sync"
)

// Responser is an interface representing the ability to execute a single HTTP transaction, mocking the Response for a given Request.
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

func (rr *Response) MatchTimes() bool {
	return rr.expectedTimes == MockUnlimitedTimes || rr.expectedTimes == rr.invokedTimes
}

func (rr *Response) Times() (expected, invoked int) {
	return rr.expectedTimes, rr.invokedTimes
}

func (rr *Response) SetExpectedTimes(expected int) {
	rr.expectedTimes = expected
}

func (rr *Response) RoundTrip(r *http.Request) (*http.Response, error) {
	rr.mux.Lock()
	defer rr.mux.Unlock()

	// unlimited mock
	if rr.expectedTimes == MockUnlimitedTimes {
		rr.invokedTimes += 1

		return rr.responder.RoundTrip(r)
	}

	// direct connect when response mock times is reach
	if rr.invokedTimes == rr.expectedTimes {
		r.URL.Scheme = rr.scheme

		return httpDefaultResponder.RoundTrip(r)
	}

	rr.invokedTimes += 1

	return rr.responder.RoundTrip(r)
}
