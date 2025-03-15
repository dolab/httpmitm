package httpmitm

import (
	"bytes"
	"net/url"
	"testing"
	"time"

	"github.com/golib/assert"
)

func Test_Testdata(t *testing.T) {
	it := assert.New(t)

	rw := bytes.NewBufferString("")

	td := NewTestdata(rw)
	it.Implements((*Testdataer)(nil), td)
}

func Test_NewTestdataFromIface(t *testing.T) {
	it := assert.New(t)

	// should work with Testdataer
	rv := bytes.NewReader([]byte{})

	reader, err := NewTestdataFromIface(rv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)

	// should work with url.Values
	vv := url.Values{"": []string{}}

	reader, err = NewTestdataFromIface(vv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)

	// should work with string
	sv := ""

	reader, err = NewTestdataFromIface(sv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)

	// should work with byte
	bv := []byte{}

	reader, err = NewTestdataFromIface(bv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)

	// error with unsupported type
	iv := time.Now()

	reader, err = NewTestdataFromIface(iv)
	it.EqualError(ErrUnsupported, err.Error())
	it.Nil(reader)
}

func Test_NewJsonTestdataFromIface(t *testing.T) {
	it := assert.New(t)

	// should work with unsupported types
	iv := time.Now()

	reader, err := NewJsonTestdataFromIface(iv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)
}

func Test_NewXmlTestdataFromIface(t *testing.T) {
	it := assert.New(t)

	// should work with unsupported types
	iv := time.Now()

	reader, err := NewXmlTestdataFromIface(iv)
	it.Nil(err)
	it.Implements((*Testdataer)(nil), reader)
}
