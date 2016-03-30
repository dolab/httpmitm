package httpmitm

import (
	"net/http"
	"net/url"
	"sync"
)

var (
	RefusedResponser = NewResponser(NewRefusedResponder(), MockWildcard, MockUnlimitedTimes)
	TimeoutResponser = NewResponser(NewTimeoutResponder(), MockWildcard, MockUnlimitedTimes)
)

// Responser is an container of mocks for the same method and domain
type Responser struct {
	mux sync.RWMutex

	mocks map[string]*Mocker // relates request path with mocker under the same domain
}

// NewResponser creates a new *Responser and adds a new mokcer with rawurl's path
func NewResponser(responder http.RoundTripper, rawurl string, times int) *Responser {
	r := &Responser{
		mocks: make(map[string]*Mocker),
	}

	return r.New(responder, rawurl, times)
}

// New adds new mocker to *Responser with rawurl's path.
// NOTE: it may overwrite existed mocker with the same parsed path.
func (r *Responser) New(responder http.RoundTripper, rawurl string, times int) *Responser {
	r.mux.Lock()
	defer r.mux.Unlock()

	urlobj, err := url.Parse(rawurl)
	if err != nil {
		panic(err.Error())
	}

	path := urlobj.Path
	if path == "" {
		path = "/"
	}

	r.mocks[path] = &Mocker{
		responder:     responder,
		matcher:       DefaultMatcher,
		rawurl:        rawurl,
		originScheme:  urlobj.Scheme,
		expectedTimes: times,
		invokedTimes:  0,
	}

	return r
}

// SetMatcherByRawURL changes request matcher of mocker releated with given rawurl's path
func (r *Responser) SetMatcherByRawURL(rawurl string, matcher RequestMatcher) {
	mocker := r.FindByRawURL(rawurl)
	if mocker == nil {
		panic("Unstubbed URL: " + rawurl)
	}

	mocker.SetMatcher(matcher)
}

// SetExpectedTimesByRawURL changes expected times of mocker releated with given rawurl's path
func (r *Responser) SetExpectedTimesByRawURL(rawurl string, expected int) {
	mocker := r.FindByRawURL(rawurl)
	if mocker == nil {
		panic("Unstubbed URL: " + rawurl)
	}

	mocker.SetExpectedTimes(expected)
}

// Mocks returns all mockers of the *Responser
func (r *Responser) Mocks() map[string]*Mocker {
	return r.mocks
}

// Find resolves mocker releated with the path, its using following steps:
// 	1, try wildcard, e.g. *
// 	2, try path, e.g. /user
// 	3, try /, known as root path
// NOTE: It returns mocker of the root path default if exists
func (r *Responser) Find(path string) *Mocker {
	r.mux.RLock()
	defer r.mux.RUnlock()

	// first, try wildcard
	mocker, ok := r.mocks[MockWildcard]
	if ok {
		return mocker
	}

	// second, try request path
	mocker, ok = r.mocks[path]
	if ok {
		return mocker
	}

	// fallback to root path
	return r.mocks["/"]
}

// FindByURL returns mocker of url path
func (r *Responser) FindByURL(urlobj *url.URL) *Mocker {
	return r.Find(urlobj.Path)
}

// FindByRawURL returns mocker of parsed raw url path
func (r *Responser) FindByRawURL(rawurl string) *Mocker {
	urlobj, err := url.Parse(rawurl)
	if err != nil {
		panic(err.Error())
	}

	return r.FindByURL(urlobj)
}

// RoundTrip implements http.Roundtripper
func (r *Responser) RoundTrip(req *http.Request) (*http.Response, error) {
	mocker := r.Find(req.URL.Path)
	if mocker == nil {
		return nil, ErrNotFound
	}

	return mocker.RoundTrip(req)
}
