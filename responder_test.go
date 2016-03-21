package httpmitm

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewResponder(t *testing.T) {
	assertion := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Testing":    []string{"testing"},
	}
	body := "Hello, world!"

	responder := NewResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(header, response.Header)

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(body, string(b))
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

	responder := NewResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
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

	responder := NewJsonResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(header, response.Header)

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(`{"name":"testing"}`, string(b))
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

	responder := NewJsonResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
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

	responder := NewXmlResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(header, response.Header)

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal(`<Responder xmlns="http://xmlns.example.com"><Name>testing</Name></Responder>`, string(b))
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

	responder := NewXmlResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.NotNil(err)
	assertion.Nil(response)
}

func Test_NewBsonResponder(t *testing.T) {
	assertion := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"application/bson"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		Name string `bson:"name"`
	}{"testing"}

	responder := NewBsonResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.Nil(err)
	assertion.Equal(code, response.StatusCode)
	assertion.Equal(header, response.Header)

	b, _ := ioutil.ReadAll(response.Body)
	response.Body.Close()
	assertion.Equal([]byte{0x17, 0x0, 0x0, 0x0, 0x2, 0x6e, 0x61, 0x6d, 0x65, 0x0, 0x8, 0x0, 0x0, 0x0, 0x74, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x67, 0x0, 0x0}, b)
}

func Test_NewBsonResponderWithError(t *testing.T) {
	assertion := assert.New(t)
	code := 200
	header := http.Header{
		"Content-Type": []string{"application/bson"},
		"X-Testing":    []string{"testing"},
	}
	body := struct {
		Ch chan<- bool `json:"channel"`
	}{make(chan<- bool, 1)}

	responder := NewBsonResponder(code, header, body)
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.NotNil(err)
	assertion.Nil(response)
}

func Test_RefuseResponse(t *testing.T) {
	assertion := assert.New(t)

	assertion.Implements((*Responser)(nil), RefuseResponse)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := RefuseResponse.RoundTrip(request)
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

	responder := NewCalleeResponder(func(r *http.Request) (int, http.Header, []byte, error) {
		return code, header, []byte(body), nil
	})
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
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

	responder := NewCalleeResponder(func(r *http.Request) (int, http.Header, []byte, error) {
		return code, header, []byte(body), ErrUnsupport
	})
	assertion.Implements((*Responser)(nil), responder)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	response, err := responder.RoundTrip(request)
	assertion.EqualError(ErrUnsupport, err.Error())
	assertion.Nil(response)
}
