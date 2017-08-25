package httpmitm

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	"github.com/golib/assert"
)

func Test_Testdata(t *testing.T) {
	assertion := assert.New(t)

	rw := bytes.NewBufferString("")

	td := NewTestdata(rw)
	assertion.Implements((*Testdataer)(nil), td)
}

func Test_NewTestdataFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with Testdataer
	rv := bytes.NewReader([]byte{})

	reader, err := NewTestdataFromIface(rv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)

	// should work with url.Values
	vv := url.Values{"": []string{}}

	reader, err = NewTestdataFromIface(vv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)

	// should work with string
	sv := ""

	reader, err = NewTestdataFromIface(sv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)

	// should work with byte
	bv := []byte{}

	reader, err = NewTestdataFromIface(bv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)

	// error with unsupported type
	iv := time.Now()

	reader, err = NewTestdataFromIface(iv)
	assertion.EqualError(ErrUnsupport, err.Error())
	assertion.Nil(reader)
}

func Test_NewJsonTestdataFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with unspported types
	iv := time.Now()

	reader, err := NewJsonTestdataFromIface(iv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)
}

func Test_NewXmlTestdataFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with unspported types
	iv := time.Now()

	reader, err := NewXmlTestdataFromIface(iv)
	assertion.Nil(err)
	assertion.Implements((*Testdataer)(nil), reader)
}
