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
	testing          *testing.T
}

func NewMitmTransport() *MitmTransport {
	return &MitmTransport{
		mockedResponses: make(map[string]*Response),
		lastMocked:      false,
		lastMockedKey:   "",
		lastMockedTimes: MockDefaultTimes,
		stubbed:         false,
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

	for key, response := range mt.mockedResponses {
		// does not match invoke?
		if !response.MatchTimes() {
			mt.testing.Fail()

			_, file, line, _ := runtime.Caller(1)
			expected, invoked := response.Times()

			fmt.Printf(`        Error Trace:    %s:%d
        Error:          Expected invoke %s with %d times, but got %d times

`, filepath.Base(file), line, key, expected, invoked)
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

	mockedKey := method + ":" + rawurl

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

func (mt *MitmTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// direct connect for none mitm scheme
	if strings.ToLower(r.URL.Scheme) != MockScheme {
		return httpDefaultResponder.RoundTrip(r)
	}

	// case-insensitive url
	rawurl := strings.ToLower(r.URL.String())

	response, ok := mt.mockedResponses[r.Method+":"+rawurl]
	if !ok {
		// fallback to abs path
		if r.URL.RawQuery != "" {
			response, ok = mt.mockedResponses[r.Method+":"+strings.SplitN(rawurl, "?", 2)[0]]
		}
	}

	if ok {
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
