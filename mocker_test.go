package httpmitm

import (
	"net/http"
	"testing"

	"github.com/golib/assert"
)

func Test_NewMocker(t *testing.T) {
	assertion := assert.New(t)

	responder := new(testResponserRounderTrip)
	rawurl := mockURL
	times := 1

	mocker := NewMocker(responder, rawurl, times)
	assertion.Implements((*http.RoundTripper)(nil), mocker)
	assertion.False(mocker.IsTimesExceed())
	assertion.Equal(times, mocker.expectedTimes)
	assertion.Equal(0, mocker.invokedTimes)

	// invocation
	request, _ := http.NewRequest("GET", rawurl, nil)
	assertion.True(mocker.IsRequestMatched(request))
	assertion.Equal(times, mocker.expectedTimes)

	mocker.RoundTrip(request)

	assertion.False(mocker.IsTimesExceed())
	assertion.Equal(1, mocker.invokedTimes)

	// exceed
	mocker.RoundTrip(request)

	assertion.True(mocker.IsTimesExceed())
	assertion.Equal(2, mocker.invokedTimes)
}
