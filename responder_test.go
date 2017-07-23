package httpmitm

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/golib/assert"
)

func Test_NewResponder(t *testing.T) {
	assertion := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewResponder(code, header, body)
	assertion.Implements((*http.RoundTripper)(nil), responder)

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(strconv.Itoa(len(body)), response.Header.Get("Content-Length"))
	assertion.NotNil(response.Request)

	b, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Nil(err)
	assertion.Equal(body, string(b))
}

func Test_NewResponderWithSuppliedContentLength(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.Equal("1024", response.Header.Get("Content-Length"))

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.NotEqual(1024, len(b))
}

func Test_NewResponderWithError(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.EqualError(ErrUnsupport, err.Error())
	assertion.Nil(response)
}

func Test_NewJsonResponder(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal("application/json", response.Header.Get("Content-Type"))
	assertion.Equal(strconv.Itoa(len(rawbody)), response.Header.Get("Content-Length"))

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(rawbody, string(b))
}

func Test_NewJsonResponderWithError(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.NotNil(err)
	assertion.Nil(response)
}

func Test_NewXmlResponder(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal("text/xml", response.Header.Get("Content-Type"))
	assertion.Equal(strconv.Itoa(len(rawbody)), response.Header.Get("Content-Length"))

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(rawbody, string(b))
}

func Test_NewXmlResponderWithError(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.NotNil(err)
	assertion.Nil(response)
}

func Test_NewCalleeResponder(t *testing.T) {
	assertion := assert.New(t)
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
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(header, response.Header)

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(body, string(b))
}

func Test_NewCalleeResponderWithError(t *testing.T) {
	assertion := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"
	rawurl := mockURL

	responder := NewCalleeResponder(func(r *http.Request) (int, http.Header, io.Reader, error) {
		return code, header, strings.NewReader(body), ErrUnsupport
	})

	request, _ := http.NewRequest("GET", rawurl, nil)
	response, err := responder.RoundTrip(request)
	assertion.EqualError(ErrUnsupport, err.Error())
	assertion.Nil(response)
}
