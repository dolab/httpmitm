package httpmitm

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var (
	mockServer *httptest.Server
	mockURL    string
	stubURL    string
)

func TestMain(m *testing.M) {
	mockServer = httptest.NewServer(&server{})
	mockURL = mockServer.URL
	stubURL = "mitm" + mockURL[4:]

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
