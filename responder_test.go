package httpmitm

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/golib/assert"
)

func Test_NewResponder(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewResponder(code, header, body)
	it.Implements((*http.RoundTripper)(nil), responder)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.Nil(err)
	it.Equal(code, response.StatusCode)
	it.Equal(strconv.Itoa(len(body)), response.Header.Get("Content-Length"))
	it.NotNil(response.Request)

	b, err := io.ReadAll(response.Body)
	response.Body.Close()
	it.Nil(err)
	it.Equal(body, string(b))
}

func Test_NewResponderWithSuppliedContentLength(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type":   []string{"text/plain"},
		"Content-Length": []string{"1024"},
		"X-Testing":      []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewResponder(code, header, body)
	request, _ := http.NewRequest("GET", rawurl, nil)

	response, _ := responder.RoundTrip(request)
	it.Equal("1024", response.Header.Get("Content-Length"))

	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	it.NotEqual(1024, len(b))
}

func Test_NewResponderWithError(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		Name string
	}{"testing"}
	rawurl := mockURL

	responder := NewResponder(code, header, body)
	request, _ := http.NewRequest("GET", rawurl, nil)

	response, err := responder.RoundTrip(request)
	it.EqualError(ErrUnsupported, err.Error())
	it.Nil(response)
}

func Test_NewJsonResponder(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		Name string `json:"name"`
	}{"testing"}
	rawurl := mockURL
	rawbody := `{"name":"testing"}`

	responder := NewJsonResponder(code, header, body)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.Nil(err)
	it.Equal(code, response.StatusCode)
	it.Equal("application/json", response.Header.Get("Content-Type"))
	it.Equal(strconv.Itoa(len(rawbody)), response.Header.Get("Content-Length"))

	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	it.Equal(rawbody, string(b))
}

func Test_NewJsonResponderWithError(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		Ch chan<- bool `json:"channel"`
	}{make(chan<- bool, 1)}
	rawurl := mockURL

	responder := NewJsonResponder(code, header, body)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.NotNil(err)
	it.Nil(response)
}

func Test_NewXmlResponder(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		XMLName xml.Name
		Name    string `xml:"Name"`
	}{
		XMLName: xml.Name{
			Space: "http://xmlns.example.com",
			Local: "Responder",
		},
		Name: "testing",
	}
	rawurl := mockURL
	rawbody := `<Responder xmlns="http://xmlns.example.com"><Name>testing</Name></Responder>`

	responder := NewXmlResponder(code, header, body)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.Nil(err)
	it.Equal(code, response.StatusCode)
	it.Equal("text/xml", response.Header.Get("Content-Type"))
	it.Equal(strconv.Itoa(len(rawbody)), response.Header.Get("Content-Length"))

	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	it.Equal(rawbody, string(b))
}

func Test_NewXmlResponderWithError(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		XMLName xml.Name
		Ch      chan<- bool `xml:"Channel"`
	}{
		XMLName: xml.Name{
			Space: "http://xmlns.example.com",
			Local: "Responder",
		},
		Ch: make(chan<- bool, 1),
	}
	rawurl := mockURL

	responder := NewXmlResponder(code, header, body)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.NotNil(err)
	it.Nil(response)
}

func Test_NewCalleeResponder(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewCalleeResponder(func(r *http.Request) (int, http.Header, io.Reader, error) {
		return code, header, strings.NewReader(body), nil
	})

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.Nil(err)
	it.Equal(code, response.StatusCode)
	it.Equal(header, response.Header)

	b, _ := io.ReadAll(response.Body)
	response.Body.Close()
	it.Equal(body, string(b))
}

func Test_NewCalleeResponderWithError(t *testing.T) {
	it := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewCalleeResponder(func(r *http.Request) (int, http.Header, io.Reader, error) {
		return code, header, strings.NewReader(body), ErrUnsupported
	})

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	it.EqualError(ErrUnsupported, err.Error())
	it.Nil(response)
}
