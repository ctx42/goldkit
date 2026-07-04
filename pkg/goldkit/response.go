package goldkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
	"gopkg.in/yaml.v3"
)

// Response represents HTTP response golden file.
type Response struct {
	*base      `yaml:",inline"`
	StatusCode int      `yaml:"statusCode"`
	Headers    []string `yaml:"headers"`

	headers http.Header // Parsed Headers field.
	body    Body        // Parsed golden file body field.
	t       tester.T    // Test manager.
}

// NewResponse creates a new [Response] object based on the YAML golden file
// represented by [Source]. The golden file must have a root field named
// "response" which is used to set field values for the [Response] object. On
// error, it marks the test as failed and returns nil.
//
// Example YAML file:
//
//	response:
//	  statusCode: 200
//	  headers:
//	    - 'Authorization: Bearer token'
//	    - 'Content-Type: application/json'
//	  meta:
//	    key1: val1
//	    key2: 123
//	    key3: 12.3
//	    key4: 2021-02-28T10:24:25.123Z
//	  bodyType: json
//	  body: |
//	      { "key2": "val2" }
func NewResponse(t tester.T, src Source) *Response {
	t.Helper()

	data, err := io.ReadAll(src)
	if err != nil {
		t.Error(err)
		return nil
	}

	// Response golden file has one root field 'response', so we create
	// a simple wrap and set default field values.
	wrap := struct {
		Response *Response `yaml:"response"`
	}{
		Response: &Response{
			base: &base{
				BodyType: Text,
			},
			t: t,
		},
	}

	if err = yaml.Unmarshal(data, wrap); err != nil {
		t.Error(err)
		return nil
	}

	rsp := wrap.Response
	if err = rsp.setup(src.Path); err != nil {
		t.Error(err)
		return nil
	}
	return rsp
}

// Response returns a [http.Response] object based on [Response].
func (rsp *Response) Response() *http.Response {
	r := &http.Response{Header: make(http.Header)}
	code := rsp.StatusCode
	r.StatusCode = code
	r.Status = fmt.Sprintf("%d %s", code, http.StatusText(code))
	r.Header = maps.Clone(rsp.headers)
	// Derive the Content-Type from the body type unless the golden file set
	// it explicitly, mirroring how [Request] fills the request header.
	if _, ok := r.Header["Content-Type"]; !ok {
		rsp.body.SetContentTypeHeader(r.Header)
	}
	r.Body = io.NopCloser(bytes.NewReader(rsp.Body()))
	return r
}

// Body returns a copy of the response's body as a byte slice.
func (rsp *Response) Body() []byte {
	return rsp.body.Body()
}

// Assert asserts response matches the golden file. Returns true on success,
// otherwise it marks the test as failed and returns false.
//
// All headers defined in the golden file must match exactly, but the "have"
// response may have more headers than defined in the golden file.
//
// To compare response bodies, a method best suited for body type is used.
// For example, when comparing JSON bodies, both byte slices don't have to be
// identical, but they must represent the same data.
func (rsp *Response) Assert(have *http.Response) bool {
	rsp.t.Helper()

	if rsp.StatusCode != have.StatusCode {
		msg := notice.New("expected response status code to be equal").
			Want("%d", rsp.StatusCode).
			Have("%d", have.StatusCode)
		rsp.t.Error(msg)
		return false
	}

	if !assert.MapSubset(rsp.t, rsp.headers, have.Header) {
		return false
	}

	haveBody, rc := cloneReader(rsp.t, have.Body)
	defer func() { have.Body = rc }()
	return rsp.body.Assert(rsp.t, haveBody)
}

// setup takes a path to the golden file and sets up the [Response]. The method
// must be called after unmarshalling the [Response].
func (rsp *Response) setup(pth string) error {
	var err error

	// Parse request body based on the body type field.
	rsp.body, err = parseBody(pth, rsp.RawBody, rsp.BodyType)
	if err != nil {
		return err
	}

	if rsp.headers, err = lines2Headers(rsp.Headers...); err != nil {
		return err
	}
	return rsp.validate()
}

// validate validates response loaded from golden file.
func (rsp *Response) validate() error {
	if rsp.StatusCode == 0 {
		return errors.New("HTTP response status code field is required")
	}
	return nil
}
