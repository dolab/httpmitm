package httpmitm

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// MitmTransport implements http.RoundTripper, which hijacks http request issued by an http.Client with mitm scheme.
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
		lastMockedMethod:  "",
		lastMockedURL:     "",
		lastMockedMatcher: DefaultMatcher,
		lastMockedTimes:   MockDefaultTimes,
	}
}

// StubDefaultTransport stubs http.DefaultTransport with MitmTransport.
func (mitm *MitmTransport) StubDefaultTransport(t *testing.T) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	if !mitm.stubbed {
		mitm.stubbed = true

		http.DefaultTransport = mitm
	}

	mitm.testing = t

	return mitm
}

// UnstubDefaultTransport restores http.DefaultTransport
func (mitm *MitmTransport) UnstubDefaultTransport() {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	if mitm.stubbed {
		mitm.stubbed = false

		http.DefaultTransport = httpDefaultResponder
	}

	// is times miss match?
	if !mitm.paused {
		errlogs := []string{}
		for key, response := range mitm.stubs {
			for path, mocker := range response.Mocks() {
				if !mocker.IsTimesMatched() {
					key = strings.Replace(key, MockScheme, mocker.Scheme(), 1)
					expected, invoked := mocker.Times()

					errlogs = append(errlogs, "        Error Trace:    %s:%d\n        Error:          Expected "+key+path+" with "+fmt.Sprintf("%d", expected)+" times, but got "+fmt.Sprintf("%d", invoked)+" times\n\n")
				}
			}
		}

		if len(errlogs) > 0 {
			pcs := make([]uintptr, 10)
			max := runtime.Callers(2, pcs)

			var (
				pcfunc *runtime.Func
				pcfile string
				pcline int
			)
			for i := 0; i < max; i++ {
				pcfunc = runtime.FuncForPC(pcs[i] - 1)
				if strings.HasPrefix(pcfunc.Name(), "runtime.") {
					continue
				}

				pcfile, pcline = pcfunc.FileLine(pcs[i])
			}

			// format errlogs
			for i, errlog := range errlogs {
				errlogs[i] = fmt.Sprintf(errlog, filepath.Base(pcfile), pcline)
			}

			fmt.Printf("--- FAIL: %s\n%s", pcfunc.Name(), strings.Join(errlogs, "\n"))
			mitm.testing.Fail()
		}
	}

	mitm.testing = nil
	mitm.stubs = make(map[string]*Responser)
}

// MockRequest stubs resource with request method
func (mitm *MitmTransport) MockRequest(method, rawurl string) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	key, err := mitm.calcRequestKey(method, rawurl)
	if err != nil {
		panic(err.Error())
	}

	// adjust empty responder with RefusedResponser for prev un-finished mock
	if mitm.mocked == false && mitm.lastMockedMethod != "" && mitm.lastMockedURL != "" {
		lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)
		if lastKey == key {
			return mitm
		}

		if mitm.stubs[lastKey] == nil {
			mitm.stubs[lastKey] = RefusedResponser
		}
	}

	mitm.mocked = false
	mitm.lastMockedMethod = method
	mitm.lastMockedURL = rawurl
	mitm.lastMockedMatcher = DefaultMatcher
	mitm.lastMockedTimes = MockDefaultTimes

	return mitm
}

// ByMatcher apply custom matcher for current stub
func (mitm *MitmTransport) ByMatcher(matcher func(r *http.Request, urlobj *url.URL) bool) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	mitm.ensureChained()

	// modify mocked matcher
	if mitm.mocked {
		lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)
		mitm.stubs[lastKey].SetMatcherByRawURL(mitm.lastMockedURL, matcher)
	} else {
		mitm.lastMockedMatcher = matcher
	}

	return mitm
}

// Times apply custom match times for current stub
func (mitm *MitmTransport) Times(i int) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	mitm.ensureChained()

	if i < 0 && i != MockUnlimitedTimes {
		panic(ErrTimes.Error())
	}

	// modify mocked times
	if mitm.mocked {
		lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)
		mitm.stubs[lastKey].SetExpectedTimesByRawURL(mitm.lastMockedURL, i)
	} else {
		mitm.lastMockedTimes = i
	}

	return mitm
}

func (mitm *MitmTransport) AnyTimes() *MitmTransport {
	return mitm.Times(MockUnlimitedTimes)
}

func (mitm *MitmTransport) WithResponser(responder http.RoundTripper) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	mitm.ensureChained()

	key, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)
	if mitm.stubs[key] == nil || mitm.stubs[key] == RefusedResponser {
		mitm.stubs[key] = NewResponser(responder, mitm.lastMockedURL, mitm.lastMockedTimes)
	} else {
		mitm.stubs[key].New(responder, mitm.lastMockedURL, mitm.lastMockedTimes)
	}

	mitm.stubs[key].SetMatcherByRawURL(mitm.lastMockedURL, mitm.lastMockedMatcher)
	mitm.mocked = true

	return mitm
}

func (mitm *MitmTransport) WithResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewResponder(code, header, body))
}

func (mitm *MitmTransport) WithJsonResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewJsonResponder(code, header, body))
}

func (mitm *MitmTransport) WithXmlResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewXmlResponder(code, header, body))
}

func (mitm *MitmTransport) WithCalleeResponse(callee func(r *http.Request) (code int, header http.Header, body io.Reader, err error)) *MitmTransport {
	return mitm.WithResponser(NewCalleeResponder(callee))
}

func (mitm *MitmTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// direct connection for none mitm scheme
	if strings.ToLower(req.URL.Scheme) != MockScheme {
		return httpDefaultResponder.RoundTrip(req)
	}

	response, ok := mitm.stubs[mitm.normalizeKey(req.Method, MockScheme, req.URL.Host)]
	if !ok {
		return RefusedResponser.RoundTrip(req)
	}

	mocker := response.Find(req.URL.Path)
	if mocker == nil {
		return RefusedResponser.RoundTrip(req)
	}

	// direct connection for paused
	if mitm.paused {
		// adjust request url scheme
		req.URL.Scheme = mocker.Scheme()

		return httpDefaultResponder.RoundTrip(req)
	}

	return mocker.RoundTrip(req)
}

// TODO: what's behavior of request timeout?
func (mitm *MitmTransport) CancelRequest(r *http.Request) {

}

// Pause pauses all stubs of all requests
func (mitm *MitmTransport) Pause() {
	mitm.mux.Lock()
	if mitm.stubbed {
		mitm.paused = true
	}
	mitm.mux.Unlock()
}

// Resume resumes all paused stubs of all requests
func (mitm *MitmTransport) Resume() {
	mitm.mux.Lock()
	if mitm.stubbed {
		mitm.paused = false
	}
	mitm.mux.Unlock()
}

func (mitm *MitmTransport) ensureChained() {
	if mitm.lastMockedMethod == "" || mitm.lastMockedURL == "" {
		panic(ErrInvocation.Error())
	}
}

func (mitm *MitmTransport) calcRequestKey(method, rawurl string) (string, error) {
	urlobj, err := url.Parse(rawurl)
	if err != nil {
		return "", err
	}

	return mitm.normalizeKey(method, MockScheme, urlobj.Host), nil
}

func (mitm *MitmTransport) normalizeKey(method, scheme, host string) string {
	return strings.ToUpper(method) + " " + strings.TrimRight(strings.ToLower(scheme+"://"+host), "/")
}
