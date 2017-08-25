package httpmitm

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

var (
	mockServer *httptest.Server
	mockURL    string
	stubURL    string
	apidata    *testdata
)

func TestMain(m *testing.M) {
	mockServer = httptest.NewServer(&server{})
	mockURL = mockServer.URL
	stubURL = "mitm" + mockURL[4:]
	apidata = newTestdata(map[string][]byte{
		"GET /": []byte("Hello, httpmitm!"),
	})

	code := m.Run()

	// shutdown mock server
	mockServer.Close()

	os.Exit(code)
}

type server struct{}

func (srv *server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)

	switch req.URL.Path {
	case "/mock":
		resp.Write([]byte(strings.ToUpper(req.Method + " MOCK OK")))

	case "/httpmitm":
		resp.Header().Set("X-Http-Mitm", "true")
		resp.Write([]byte(strings.ToUpper(req.Method + " OK")))

	default:
		resp.Write([]byte(strings.ToUpper(req.Method + " OK")))

	}
}

// example
type testdata struct {
	contents map[string][]byte
}

func newTestdata(data map[string][]byte) *testdata {
	return &testdata{
		contents: data,
	}
}

func (td *testdata) Key(method string, urlobj *url.URL) (key string) {
	abspath := urlobj.Path
	if abspath == "" {
		abspath = "/"
	}

	return fmt.Sprintf("%s %s", method, abspath)
}

func (td *testdata) Read(key string) (data []byte, err error) {
	data, ok := td.contents[key]
	if !ok {
		err = errors.New("Not found")
	}

	return
}

func (td *testdata) Write(key string, data []byte) (err error) {
	td.contents[key] = data

	return nil
}
