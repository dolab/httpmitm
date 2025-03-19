package httpmitm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// MitmTransport implements http.RoundTripper, which hijacks http request issued by an http.Client with mitm scheme.
// It deferred to the registered responders instead of making a real http request.
type MitmTransport struct {
	mux sync.Mutex

	testing *testing.T

	stubs   map[string]*Responser // responders registered for MITM request
	stubbed atomic.Bool           // indicate whether http.DefaultTransport stubbed?
	paused  atomic.Bool           // indicate whether current mocked transport paused?
	mocked  atomic.Bool           // indicate whether current chain finished?

	lastMockedMethod  string
	lastMockedURL     string
	lastMockedMatcher RequestMatcher
	lastMockedTimes   int
}

// NewMitmTransport creates MitmTransport for stubs && mocks.
func NewMitmTransport() *MitmTransport {
	return &MitmTransport{
		stubs:             make(map[string]*Responser),
		lastMockedMethod:  "",
		lastMockedURL:     "",
		lastMockedMatcher: DefaultMatcher,
		lastMockedTimes:   MockDefaultTimes,
	}
}

// StubDefaultTransport stubs http.DefaultTransport with MitmTransport.
func (mitm *MitmTransport) StubDefaultTransport(t *testing.T) *MitmTransport {
	mitm.testing = t

	if !mitm.stubbed.Swap(true) {
		http.DefaultTransport = mitm
	}

	return mitm
}

// UnstubDefaultTransport restores http.DefaultTransport
func (mitm *MitmTransport) UnstubDefaultTransport() {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	if mitm.stubbed.Swap(false) {
		http.DefaultTransport = httpDefaultResponder
	}

	// is times missing match?
	if !mitm.paused.Load() {
		var errlogs []string

		for key, stubs := range mitm.stubs {
			for path, mocker := range stubs.Mocks() {
				if mocker.IsTimesExceed() {
					key = strings.Replace(key, MockScheme, mocker.Scheme(), 1)
					expected, invoked := mocker.Times()

					errlogs = append(errlogs, DefaultLeaddingSpace+"Error Trace:    %s:%d\n"+DefaultLeaddingSpace+"Error:          Expected "+key+path+" with "+fmt.Sprintf("%d", expected)+" times, but got "+fmt.Sprintf("%d", invoked)+" times\n")
				}
			}
		}

		if len(errlogs) > 0 {
			pcs := make([]uintptr, 20)
			frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])

			var (
				frame runtime.Frame
				more  bool
			)
			for {
				tmpframe, tmpmore := frames.Next()
				if strings.HasPrefix(tmpframe.Function, "testing.") {
					if !tmpmore {
						frame, more = tmpframe, tmpmore
					}

					break
				}

				frame, more = tmpframe, tmpmore
				if !more {
					break
				}
			}

			// format errlogs
			for i, errlog := range errlogs {
				errlogs[i] = fmt.Sprintf(errlog, filepath.Base(frame.File), frame.Line)
			}

			fmt.Printf("--- FAIL: %s\n%s\n", filepath.Base(frame.Function), strings.Join(errlogs, "\n"))
			mitm.testing.Fail()
		}
	}

	mitm.stubs = make(map[string]*Responser)
	mitm.testing = nil
}

// MockRequest stubs resource with request method
func (mitm *MitmTransport) MockRequest(method, rawurl string) *MitmTransport {
	mitm.mux.Lock()
	defer mitm.mux.Unlock()

	key, err := mitm.calcRequestKey(method, rawurl)
	if err != nil {
		panic(err.Error())
	}

	// adjust empty responder with RefusedResponser for prev un-finished mocks
	if !mitm.mocked.Load() && mitm.lastMockedMethod != "" && mitm.lastMockedURL != "" {
		lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)
		if lastKey == key {
			return mitm
		}

		if mitm.stubs[lastKey] == nil {
			mitm.stubs[lastKey] = RefusedResponser
		}
	}

	mitm.mocked.Store(false)
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
	lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)

	responser, ok := mitm.stubs[lastKey]
	if ok {
		responser.SetMatcherByRawURL(mitm.lastMockedURL, matcher)
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
	lastKey, _ := mitm.calcRequestKey(mitm.lastMockedMethod, mitm.lastMockedURL)

	responser, ok := mitm.stubs[lastKey]
	if ok {
		responser.SetExpectedTimesByRawURL(mitm.lastMockedURL, i)
	} else {
		mitm.lastMockedTimes = i
	}

	return mitm
}

// AnyTimes apply ulimited times for current stub
func (mitm *MitmTransport) AnyTimes() *MitmTransport {
	return mitm.Times(MockUnlimitedTimes)
}

// WithResponser apply http.RoundTripper for current stub
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

	mitm.mocked.Store(true)

	return mitm
}

// WithResponse apply http text/palin response for current stub
func (mitm *MitmTransport) WithResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewResponder(code, header, body))
}

// WithJsonResponse apply http application/json response for current stub
func (mitm *MitmTransport) WithJsonResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewJsonResponder(code, header, body))
}

// WithXmlResponse apply http text/xml response for current stub
func (mitm *MitmTransport) WithXmlResponse(code int, header http.Header, body interface{}) *MitmTransport {
	return mitm.WithResponser(NewXmlResponder(code, header, body))
}

// WithCalleeResponse apply custom func for current stub
func (mitm *MitmTransport) WithCalleeResponse(callee func(r *http.Request) (code int, header http.Header, body io.Reader, err error)) *MitmTransport {
	return mitm.WithResponser(NewCalleeResponder(callee))
}

// RoundTrip implments http.RoundTripper
func (mitm *MitmTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// direct connection for none mitm scheme
	if strings.ToLower(r.URL.Scheme) != MockScheme {
		return httpDefaultResponder.RoundTrip(r)
	}

	response, ok := mitm.stubs[mitm.normalizeKey(r.Method, MockScheme, r.URL.Host)]
	if !ok {
		return RefusedResponser.RoundTrip(r)
	}

	mocker := response.Find(r.URL.Path)
	if mocker == nil {
		return NotFoundResponser.RoundTrip(r)
	}

	// direct connection for paused
	if mitm.paused.Load() {
		// adjust request url scheme
		r.URL.Scheme = mocker.Scheme()

		resp, err := httpDefaultResponder.RoundTrip(r)
		if err != nil {
			return resp, err
		}

		// try to write back response data if response code is 2xx or equal to expected
		responder, ok := mocker.responder.(*Responder)
		if !ok || (resp.StatusCode/100 != 2 && resp.StatusCode != responder.code) {
			return resp, err
		}

		var (
			data []byte
		)

		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			gzipReader, gzipErr := gzip.NewReader(resp.Body)
			if gzipErr != nil {
				return resp, gzipErr
			}

			data, err = io.ReadAll(gzipReader)
			if err != nil {
				return resp, err
			}
			gzipReader.Close()

			// reset response header with new data
			resp.Header.Del("Content-Encoding")
			resp.Header.Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))

		default:
			data, err = io.ReadAll(resp.Body)
			if err != nil {
				return resp, err
			}
			resp.Body.Close()

		}

		// rewrite response body for client
		resp.Body = io.NopCloser(bytes.NewBuffer(data))

		// invoke testdata writer
		if werr := responder.Write(r.Method, r.URL, data); werr != nil {
			mitm.testing.Logf("Response writes %s %s with: %v", r.Method, r.URL.String(), werr)
		} else {
			mitm.testing.Logf("Response write %s %s OK!", r.Method, r.URL.String())
		}

		return resp, err
	}

	return mocker.RoundTrip(r)
}

// CancelRequest close request mocked
// TODO: what's behavior of request timeout?
func (mitm *MitmTransport) CancelRequest(r *http.Request) {

}

// Pause pauses all stubs of all requests
func (mitm *MitmTransport) Pause() {
	if mitm.stubbed.Load() {
		mitm.paused.Store(true)
	}
}

// Resume resumes all paused stubs of all requests
func (mitm *MitmTransport) Resume() {
	if mitm.stubbed.Load() {
		mitm.paused.Store(false)
	}
}

// PrettyPrint dumps MitmTransport in well format.
func (mitm *MitmTransport) PrettyPrint() {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("stubs<map[string]&httpmitm.Responder>{\n")
	for key, stub := range mitm.stubs {
		buf.WriteString(`    "` + key + `": &httpmitm.Responder{` + "\n")
		buf.WriteString(`        mocks<map[string]&httpmitm.Mocker>{` + "\n")
		for subkey, mock := range stub.mocks {
			tmp := fmt.Sprintf("%#v", mock)
			tmp = strings.Replace(tmp, `sync.Mutex{state:0, sema:0x0}`, "sync.Mutex()", -1)
			tmp = strings.Replace(tmp, "{", "{\n                ", -1)
			tmp = strings.Replace(tmp, ", ", ",\n                ", -1)
			tmp = strings.Replace(tmp, "}", "\n            }", -1)
			tmp = strings.Replace(tmp, "sync.Mutex()", `sync.Mutex{state:0, sema:0x0}`, -1)

			buf.WriteString(`            "` + subkey + `": ` + tmp + "\n")
		}
		buf.WriteString(`        }` + "\n")
		buf.WriteString(`    }` + "\n")
	}
	buf.WriteString("\n}\n")

	println(buf.String())
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
