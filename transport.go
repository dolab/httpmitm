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

const (
	MockScheme         = "mitm"
	MockDefaultTimes   = 1
	MockUnlimitedTimes = -1
)

// MitmTransport implements http.RoundTripper, which hijacks http requests issued by
// an http.Client with mitm scheme.
// It defferrs to the registered responders instead of making a real http request.
type MitmTransport struct {
	mux sync.Mutex

	mockedResponses map[string]*Response // responders registered for MITM request
	defaultResponse *Response            // default responder for MITM request

	lastMocked       bool // indicate whether current chain finished?
	lastMockedKey    string
	lastMockedTimes  int
	lastMockedScheme string
	stubbed          bool
	paused           bool
	testing          *testing.T
}

func NewMitmTransport() *MitmTransport {
	return &MitmTransport{
		mockedResponses: make(map[string]*Response),
		lastMocked:      false,
		lastMockedKey:   "",
		lastMockedTimes: MockDefaultTimes,
		stubbed:         false,
		paused:          false,
	}
}

// SetDefaultResponder sets default responder for all unregistered mitm request.
func (mt *MitmTransport) SetDefaultResponder(responder Responser) {
	mt.defaultResponse = NewResponse(responder, MockScheme, MockUnlimitedTimes)
}

// StubDefaultTransport stubs http.DefaultTransport with MitmTransport.
func (mt *MitmTransport) StubDefaultTransport(t *testing.T) {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if !mt.stubbed {
		mt.stubbed = true

		http.DefaultTransport = mt
	}

	mt.testing = t
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
		for key, response := range mt.mockedResponses {
			if !response.MatchTimes() {
				expected, invoked := response.Times()

				errlogs = append(errlogs, "        Error Trace:    %s:%d\n        Error:          Expected request "+key+" with "+fmt.Sprintf("%d", expected)+" times, but got "+fmt.Sprintf("%d", invoked)+" times\n\n")
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
}

func (mt *MitmTransport) MockRequest(method, rawurl string) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	// uppercase method
	method = strings.ToUpper(method)

	// case-insensitive url
	rawurl = strings.ToLower(rawurl)

	// adjust mock scheme of http:// and https://
	if !strings.HasPrefix(rawurl, MockScheme+"://") {
		urlobj, err := url.Parse(rawurl)
		if err != nil {
			return nil
		}

		urlobj.Scheme, mt.lastMockedScheme = MockScheme, urlobj.Scheme

		rawurl = urlobj.String()
	}

	mockedKey := strings.TrimRight(method+":"+rawurl, "/")

	// adjust empty responder with RefusedResponder
	if mt.lastMocked == false && mt.lastMockedKey != "" {
		if mt.lastMockedKey == mockedKey {
			return mt
		}

		if mt.mockedResponses[mt.lastMockedKey] == nil {
			mt.mockedResponses[mt.lastMockedKey] = RefuseResponse
		}
	}

	mt.mockedResponses[mockedKey] = nil
	mt.lastMocked = false
	mt.lastMockedKey = mockedKey
	mt.lastMockedTimes = MockDefaultTimes

	return mt
}

func (mt *MitmTransport) Times(i int) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if i < 0 {
		panic("Invalid times. It must be non-negative integer value.")
	}

	if mt.lastMockedKey == "" {
		panic("Not an chained invoke. Please invoke MockRequest(method, url) first.")
	}

	if mt.lastMocked {
		// modify mocked times
		mt.mockedResponses[mt.lastMockedKey].SetExpectedTimes(i)

		// reset last mock key and times
		mt.lastMockedKey = ""
		mt.lastMockedTimes = MockDefaultTimes
		mt.lastMockedScheme = ""
	} else {
		mt.lastMockedTimes = i
	}

	return mt
}

func (mt *MitmTransport) AnyTimes() *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if mt.lastMockedKey == "" {
		panic("Not an chained invoke. Please invoke MockRequest(method, url) first.")
	}

	if mt.lastMocked {
		// modify mocked times
		mt.mockedResponses[mt.lastMockedKey].SetExpectedTimes(MockUnlimitedTimes)

		// reset last mock key and times
		mt.lastMockedKey = ""
		mt.lastMockedTimes = MockDefaultTimes
		mt.lastMockedScheme = ""
	} else {
		mt.lastMockedTimes = MockUnlimitedTimes
	}

	return mt
}

func (mt *MitmTransport) WithResponser(responder Responser) *MitmTransport {
	mt.mux.Lock()
	defer mt.mux.Unlock()

	if mt.lastMockedKey == "" {
		panic("Not an chained invoke. Please invoke MockRequest(method, url) first.")
	}

	mt.mockedResponses[mt.lastMockedKey] = NewResponse(responder, mt.lastMockedScheme, mt.lastMockedTimes)
	mt.lastMocked = true

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

func (mt *MitmTransport) WithBsonResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mt.WithResponser(NewBsonResponder(code, header, body))
}

func (mt *MitmTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// direct connect for none mitm scheme
	if strings.ToLower(r.URL.Scheme) != MockScheme {
		return httpDefaultResponder.RoundTrip(r)
	}

	// case-insensitive url
	rawurl := strings.ToLower(r.URL.String())

	response, ok := mt.mockedResponses[r.Method+":"+strings.TrimRight(rawurl, "/")]

	if !ok {
		// fallback to abs path
		if r.URL.RawQuery != "" {
			response, ok = mt.mockedResponses[r.Method+":"+strings.TrimRight(strings.SplitN(rawurl, "?", 2)[0], "/")]
		}
	}

	if ok {
		// direct connect for paused
		if mt.paused {
			r.URL.Scheme = response.scheme

			return httpDefaultResponder.RoundTrip(r)
		}

		return response.RoundTrip(r)
	}

	if mt.defaultResponse == nil {
		return RefuseResponse.RoundTrip(r)
	}

	return mt.defaultResponse.RoundTrip(r)
}

// TODO: what's timeout behavior?
func (mt *MitmTransport) CancelRequest(r *http.Request) {

}

// Pause pauses mock for all requests
func (mt *MitmTransport) Pause() {
	mt.mux.Lock()
	if mt.stubbed {
		mt.paused = true
	}
	mt.mux.Unlock()
}

// Resume resumes mock again
func (mt *MitmTransport) Resume() {
	mt.mux.Lock()
	if mt.stubbed {
		mt.paused = false
	}
	mt.mux.Unlock()
}
