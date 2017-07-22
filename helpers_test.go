package httpmitm

import (
	"bytes"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/golib/assert"
)

func Test_NewReaderFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with io.Reader
	rv := bytes.NewReader([]byte{})

	reader, err := Helpers.NewReaderFromIface(rv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)

	// should work with url.Values
	vv := url.Values{"": []string{}}

	reader, err = Helpers.NewReaderFromIface(vv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)

	// should work with string
	sv := ""

	reader, err = Helpers.NewReaderFromIface(sv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)

	// should work with byte
	bv := []byte{}

	reader, err = Helpers.NewReaderFromIface(bv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)

	// error with unsupported type
	iv := time.Now()

	reader, err = Helpers.NewReaderFromIface(iv)
	assertion.EqualError(ErrUnsupport, err.Error())
	assertion.Nil(reader)
}

func Test_NewJsonReaderFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with unspported types
	iv := time.Now()

	reader, err := Helpers.NewJsonReaderFromIface(iv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)
}

func Test_NewXmlReaderFromIface(t *testing.T) {
	assertion := assert.New(t)

	// should work with unspported types
	iv := time.Now()

	reader, err := Helpers.NewXmlReaderFromIface(iv)
	assertion.Nil(err)
	assertion.Implements((*io.Reader)(nil), reader)
}
