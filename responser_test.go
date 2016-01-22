package httpmitm

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testRounderTrip struct {
}

func (tr *testRounderTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, nil
}

func Test_NewResponse(t *testing.T) {
	assertion := assert.New(t)
	scheme := "http"
	times := 1
	responder := new(testRounderTrip)

	res := NewResponse(responder, scheme, times)
	assertion.Equal(scheme, res.Scheme())

	expected, invoked := res.Times()
	assertion.Equal(times, expected)
	assertion.Equal(0, invoked)

	// invoke
	request, _ := http.NewRequest("GET", "http://example.com", nil)
	res.RoundTrip(request)

	expected, invoked = res.Times()
	assertion.Equal(times, expected)
	assertion.Equal(1, invoked)

}
