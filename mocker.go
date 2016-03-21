package httpmitm

import (
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	MockScheme         = "mitm"
	MockDefaultTimes   = 1
	MockUnlimitedTimes = math.MinInt64
)

var (
	// DefaultMatcher is the default implementation of RequestMatcher and is used by all mocks without supplied matcher.
	// First, it compares request by fully quoted url string;
	// Second, it only compares uri by trim string after separator ? in fallback case.
	DefaultMatcher RequestMatcher = func(r *http.Request, rawurl string) bool {
		// case-insensitive
		rawurl = strings.ToLower(rawurl)

		// first, try full url
		if strings.ToLower(r.URL.String()) == rawurl {
			return true
		}

		// second, ignore query string and fragment
		urlobj, _ := url.Parse(rawurl)

		if strings.ToLower(r.URL.Scheme+"://"+r.URL.Host+strings.TrimRight(r.URL.Path, "/")) == urlobj.Scheme+"://"+urlobj.Host+strings.TrimRight(urlobj.Path, "/") {
			return true
		}

		return false
	}
)

// RequestMatcher is a callback for detecting whether request matches the mocked url
type RequestMatcher func(r *http.Request, rawurl string) bool

type mocker struct {
	mux sync.Mutex

	responder     http.RoundTripper
	rawurl        string
	matcher       RequestMatcher
	originScheme  string // origin url scheme
	expectedTimes int    // expect mocked times
	invokedTimes  int    // really mocked times
}

func NewMocker(responder http.RoundTripper, rawurl string, times int) *mocker {
	urlobj, _ := url.Parse(rawurl)

	return &mocker{
		responder:     responder,
		rawurl:        rawurl,
		matcher:       DefaultMatcher,
		originScheme:  urlobj.Scheme,
		expectedTimes: times,
		invokedTimes:  0,
	}
}

func (m *mocker) IsRequestMatched(req *http.Request) bool {
	return m.matcher(req, m.rawurl)
}

func (m *mocker) IsTimesMatched() bool {
	return m.expectedTimes == MockUnlimitedTimes || m.expectedTimes == m.invokedTimes
}

func (m *mocker) Scheme() string {
	return m.originScheme
}

func (m *mocker) Times() (expected, invoked int) {
	return m.expectedTimes, m.invokedTimes
}

func (m *mocker) SetExpectedTimes(expected int) {
	m.expectedTimes = expected
}

func (m *mocker) SetRequestMatcher(matcher RequestMatcher) {
	m.matcher = matcher
}

func (m *mocker) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	// is an unlimited mock?
	if m.expectedTimes == MockUnlimitedTimes {
		m.invokedTimes += 1

		return m.responder.RoundTrip(req)
	}

	// connect directly when response mock times is reached
	if m.invokedTimes == m.expectedTimes {
		req.URL.Scheme = m.originScheme

		return httpDefaultResponder.RoundTrip(req)
	}

	m.invokedTimes += 1

	return m.responder.RoundTrip(req)
}
