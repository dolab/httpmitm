package httpmitm

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// MitmTransport implements http.RoundTripper, which hijacks http requests issued by
// an http.Client with mitm scheme.
// It defferrs to the registered responders instead of making a real http request.
type MitmTransport struct {
	mux sync.Mutex

	testing *testing.T
	stubs   map[string]*Responser // responders registered for MITM request
	mocked  bool                  // indicate whether current chain finished?
	stubbed bool                  // indicate whether http.DefaultTransport stubbed?
	paused  bool                  // indicate whether current mocked transport paused?

	lastMockedMethod  string
	lastMockedURL     string
	lastMockedMatcher RequestMatcher
	lastMockedTimes   int
}

func NewMitmTransport() *MitmTransport {
	return &MitmTransport{
		stubs:             make(map[string]*Responser),
		mocked:            false,
		stubbed:           false,
		paused:            false,
		lastMockedURL:     "",
		lastMockedMatcher: DefaultMatcher,
		lastMockedTimes:   MockDefaultTimes,
	}
}

// StubDefaultTransport stubs http.DefaultTransport with MitmTransport.
func (mt *MitmTransport) StubDefaultTransport(t *testing.T) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if !mt.stubbed {
		mt.stubbed = true

		http.DefaultTransport = mt
	}

	mt.testing = t

	return mt
}

// UnstubDefaultTransport restores http.DefaultTransport
func (mt *MitmTransport) UnstubDefaultTransport() {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if mt.stubbed {
		mt.stubbed = false

		http.DefaultTransport = httpDefaultResponder
	}

	// does times miss match?
	if !mt.paused {
		errlogs := []string{}
		for key, response := range mt.stubs {
			for path, mocker := range response.Mocks() {
				if !mocker.IsTimesMatched() {
					expected, invoked := mocker.Times()

					errlogs = append(errlogs, "        Error Trace:    %s:%d\n        Error:          Expected "+key+path+" with "+fmt.Sprintf("%d", expected)+" times, but got "+fmt.Sprintf("%d", invoked)+" times\n\n")
				}
			}
		}

		if len(errlogs) > 0 {
			pcs := make([]uintptr, 1)
			runtime.Callers(2, pcs)

			pcfunc := runtime.FuncForPC(pcs[0])
			pcfile, pcline := pcfunc.FileLine(pcs[0])
			pcname := filepath.Base(pcfile)

			// format errlogs
			for i, errlog := range errlogs {
				errlogs[i] = fmt.Sprintf(errlog, pcname, pcline)
			}

			fmt.Printf("--- FAIL: %s\n%s", pcfunc.Name(), strings.Join(errlogs, "\n"))
			mt.testing.Fail()
		}
	}

	mt.testing = nil
	mt.stubs = make(map[string]*Responser)
}

// MockRequest stubs resource with request method
func (mt *MitmTransport) MockRequest(method, rawurl string) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	key, err := mt.calcRequestKey(method, rawurl)
	if err != nil {
		panic(err.Error())
	}

	// adjust empty responder with RefusedResponser
	if mt.mocked == false && mt.lastMockedMethod != "" && mt.lastMockedURL != "" {
		lastKey, _ := mt.calcRequestKey(mt.lastMockedMethod, mt.lastMockedURL)
		if lastKey == key {
			return mt
		}

		if mt.stubs[lastKey] == nil {
			mt.stubs[lastKey] = RefusedResponser
		}
	}

	mt.mocked = false
	mt.lastMockedMethod = method
	mt.lastMockedURL = rawurl
	mt.lastMockedMatcher = DefaultMatcher
	mt.lastMockedTimes = MockDefaultTimes

	return mt
}

func (mt *MitmTransport) ByMatcher(matcher func(r *http.Request, rawurl string) bool) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	mt.ensureChained()

	if mt.mocked {
		// modify mocked matcher
		lastKey, _ := mt.calcRequestKey(mt.lastMockedMethod, mt.lastMockedURL)
		mt.stubs[lastKey].SetRequestMatcherByRawURL(mt.lastMockedURL, matcher)

		// reset last mocked states
		mt.lastMockedMethod = ""
		mt.lastMockedURL = ""
		mt.lastMockedMatcher = DefaultMatcher
		mt.lastMockedTimes = MockDefaultTimes
	} else {
		mt.lastMockedMatcher = matcher
	}

	return mt
}

func (mt *MitmTransport) Times(i int) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	mt.ensureChained()

	if i < 0 && i != MockUnlimitedTimes {
		panic("Invalid value of times. It must be non-negative integer value.")
	}

	if mt.mocked {
		// modify mocked times
		lastKey, _ := mt.calcRequestKey(mt.lastMockedMethod, mt.lastMockedURL)
		mt.stubs[lastKey].SetExpectedTimesByRawURL(mt.lastMockedURL, i)

		// reset last mock key and times
		mt.lastMockedMethod = ""
		mt.lastMockedURL = ""
		mt.lastMockedMatcher = DefaultMatcher
		mt.lastMockedTimes = MockDefaultTimes
	} else {
		mt.lastMockedTimes = i
	}

	return mt
}

func (mt *MitmTransport) AnyTimes() *MitmTransport {
	return mt.Times(MockUnlimitedTimes)
}

func (mt *MitmTransport) WithResponser(responder http.RoundTripper) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	mt.ensureChained()

	key, _ := mt.calcRequestKey(mt.lastMockedMethod, mt.lastMockedURL)
	if mt.stubs[key] == nil || mt.stubs[key] == RefusedResponser {
		mt.stubs[key] = NewResponser(responder, mt.lastMockedURL, mt.lastMockedTimes)
	} else {
		mt.stubs[key].New(responder, mt.lastMockedURL, mt.lastMockedTimes)
	}
	mt.stubs[key].SetRequestMatcherByRawURL(mt.lastMockedURL, mt.lastMockedMatcher)
	mt.mocked = true

	return mt
}

func (mt *MitmTransport) WithResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mt.WithResponser(NewResponder(code, header, body))
}

func (mt *MitmTransport) WithJsonResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mt.WithResponser(NewJsonResponder(code, header, body))
}

func (mt *MitmTransport) WithXmlResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mt.WithResponser(NewXmlResponder(code, header, body))
}

func (mt *MitmTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// direct connection for none mitm scheme
	if strings.ToLower(req.URL.Scheme) != MockScheme {
		return httpDefaultResponder.RoundTrip(req)
	}

	response, ok := mt.stubs[mt.normalizeKey(req.Method, MockScheme, req.URL.Host)]
	if !ok {
		return RefusedResponser.RoundTrip(req)
	}

	mocker := response.Find(req.URL.Path)
	if mocker == nil {
		return RefusedResponser.RoundTrip(req)
	}

	// direct connection for paused
	if mt.paused {
		// adjust request url scheme
		req.URL.Scheme = mocker.Scheme()

		return httpDefaultResponder.RoundTrip(req)
	}

	return mocker.RoundTrip(req)
}

// TODO: what's behavior of request timeout?
func (mt *MitmTransport) CancelRequest(r *http.Request) {

}

// Pause pauses all stubs of all requests
func (mt *MitmTransport) Pause() {
	mt.mux.Lock()
	if mt.stubbed {
		mt.paused = true
	}
	mt.mux.Unlock()
}

// Resume resumes all paused stubs of all requests
func (mt *MitmTransport) Resume() {
	mt.mux.Lock()
	if mt.stubbed {
		mt.paused = false
	}
	mt.mux.Unlock()
}

func (mt *MitmTransport) ensureChained() {
	if mt.lastMockedMethod == "" || mt.lastMockedURL == "" {
		panic("Not an chained invocation. Please invoking MockRequest(method, url) first.")
	}
}

func (mt *MitmTransport) calcRequestKey(method, rawurl string) (string, error) {
	urlobj, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}

	return mt.normalizeKey(method, MockScheme, urlobj.Host), nil
}

func (mt *MitmTransport) normalizeKey(method, scheme, host string) string {
	return strings.ToUpper(method) + " " + strings.TrimRight(strings.ToLower(scheme+"://"+host), "/")
}
