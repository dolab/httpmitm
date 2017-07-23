package httpmitm

import (
	"bytes"
	"testing"

	"github.com/golib/assert"
)

func Test_Testdata(t *testing.T) {
	assertion := assert.New(t)

	rw := bytes.NewBufferString("")

	td := NewTestdata(rw)
	assertion.Implements((*Testdataer)(nil), td)
}
