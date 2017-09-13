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
	MockWildcard       = "*"
)

var (
	// DefaultMatcher is the default implementation of RequestMatcher and is used by all mocks without matcher supplied.
	// 	First, it compares request by fully quoted url string;
	// 	Second, it only compares uri by trim string after separator ? in fallback case.
	DefaultMatcher RequestMatcher = func(r *http.Request, urlobj *url.URL) bool {
		// case-insensitive

		// first, try full url
		if strings.ToLower(r.URL.String()) == strings.ToLower(urlobj.String()) {
			return true
		}

		// second, ignore query string and fragment
		if strings.ToLower(r.URL.Host+strings.TrimRight(r.URL.Path, "/")) == strings.ToLower(urlobj.Host+strings.TrimRight(urlobj.Path, "/")) {
			return true
		}

		return false
	}
)

// RequestMatcher is a callback for detecting whether request matches the mocked url
type RequestMatcher func(r *http.Request, urlobj *url.URL) bool

// Mocker defines a request with stubbed response
type Mocker struct {
	mux sync.RWMutex

	responder     http.RoundTripper
	matcher       RequestMatcher
	rawurl        string
	originScheme  string // origin url scheme
	expectedTimes int    // expect mocked times
	invokedTimes  int    // really mocked times
}

func NewMocker(responder http.RoundTripper, rawurl string, times int) *Mocker {
	urlobj, err := url.Parse(rawurl)
	if err != nil {
		panic(err.Error())
	}

	return &Mocker{
		responder:     responder,
		matcher:       DefaultMatcher,
		rawurl:        rawurl,
		originScheme:  urlobj.Scheme,
		expectedTimes: times,
		invokedTimes:  0,
	}
}

func (m *Mocker) IsRequestMatched(req *http.Request) bool {
	if m.rawurl == MockWildcard {
		return true
	}

	// parse rawurl and inject mock scheme
	urlobj, _ := url.Parse(m.rawurl)
	urlobj.Scheme = MockScheme

	return m.matcher(req, urlobj)
}

func (m *Mocker) IsTimesUnlimited() bool {
	return m.expectedTimes == MockUnlimitedTimes
}

func (m *Mocker) IsTimesExceed() bool {
	if m.IsTimesUnlimited() {
		return false
	}

	m.mux.RLock()
	defer m.mux.RUnlock()

	return m.invokedTimes > m.expectedTimes
}

func (m *Mocker) Scheme() string {
	return m.originScheme
}

func (m *Mocker) Times() (expected, invoked int) {
	return m.expectedTimes, m.invokedTimes
}

func (m *Mocker) SetMatcher(matcher RequestMatcher) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.matcher = matcher
}

func (m *Mocker) SetExpectedTimes(expected int) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.expectedTimes = expected
}

func (m *Mocker) RoundTrip(req *http.Request) (*http.Response, error) {
	// is mocked?
	m.mux.RLock()
	if !m.IsRequestMatched(req) {
		m.mux.RUnlock()

		return httpDefaultResponder.RoundTrip(req)
	}
	m.mux.RUnlock()

	m.mux.Lock()
	defer m.mux.Unlock()

	m.invokedTimes++

	switch {
	case m.IsTimesUnlimited(): // is an unlimited mocker?
		return m.responder.RoundTrip(req)

	case m.invokedTimes > m.expectedTimes: // is expected times exceed?
		if m.originScheme != "" {
			req.URL.Scheme = m.originScheme
		}

		return httpDefaultResponder.RoundTrip(req)
	}

	return m.responder.RoundTrip(req)
}
