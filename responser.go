package httpmitm

import (
	"net/http"
	"net/url"
	"sync"
)

var (
	RefusedResponser = NewResponser(NewRefusedResponder(), "/", MockUnlimitedTimes)
)

// Responser is an container of mocks for the same method and domain
type Responser struct {
	mux   sync.RWMutex
	mocks map[string]*mocker // relates request path with mocker under the same domain
}

// NewResponser creates a new *Responser and adds a new mokcer with rawurl's path
func NewResponser(responder http.RoundTripper, rawurl string, times int) *Responser {
	r := &Responser{
		mocks: make(map[string]*mocker),
	}

	return r.New(responder, rawurl, times)
}

// New adds new mocker to *Responser with rawurl's path, this may
// overwrite existed mocker with the same path.
func (r *Responser) New(responder http.RoundTripper, rawurl string, times int) *Responser {
	r.mux.Lock()
	defer r.mux.Unlock()

	urlobj, _ := url.Parse(rawurl)

	path := urlobj.Path
	if path == "" {
		path = "/"
	}

	r.mocks[path] = &mocker{
		responder:     responder,
		rawurl:        rawurl,
		matcher:       DefaultMatcher,
		originScheme:  urlobj.Scheme,
		expectedTimes: times,
		invokedTimes:  0,
	}

	return r
}

// SetExpectedTimesByRawURL changes expected times of mocker releated with given rawurl's path
func (r *Responser) SetExpectedTimesByRawURL(rawurl string, expected int) {
	mocker := r.FindByRawURL(rawurl)
	if mocker == nil {
		panic("Unstubbed resource: " + rawurl)
	}

	mocker.SetExpectedTimes(expected)
}

// SetRequestMatcherByRawURL changes request matcher of mocker releated with given rawurl's path
func (r *Responser) SetRequestMatcherByRawURL(rawurl string, matcher RequestMatcher) {
	mocker := r.FindByRawURL(rawurl)
	if mocker == nil {
		panic("Unstubbed resource: " + rawurl)
	}

	mocker.SetRequestMatcher(matcher)
}

// Mocks returns all mockers of the *Responser
func (r *Responser) Mocks() map[string]*mocker {
	return r.mocks
}

// Find resolves mocker releated with the path, it returns mocker of the root path by default
func (r *Responser) Find(path string) *mocker {
	r.mux.RLock()
	defer r.mux.RUnlock()

	// first, try request path
	mocker, ok := r.mocks[path]
	if ok {
		return mocker
	}

	// fallback to root path
	return r.mocks["/"]
}

// FindByURL returns mocker of url path
func (r *Responser) FindByURL(urlobj *url.URL) *mocker {
	return r.Find(urlobj.Path)
}

// FindByRawURL returns mocker of url path
func (r *Responser) FindByRawURL(rawurl string) *mocker {
	urlobj, err := url.Parse(rawurl)
	if err != nil {
		return nil
	}

	return r.FindByURL(urlobj)
}

func (r *Responser) RoundTrip(req *http.Request) (*http.Response, error) {
	mocker := r.Find(req.URL.Path)
	if mocker == nil {
		return nil, ErrNotFound
	}

	return mocker.RoundTrip(req)
}
