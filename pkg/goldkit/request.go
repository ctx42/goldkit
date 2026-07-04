package goldkit

import (
	"bytes"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/ctx42/testing/pkg/check"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
	"gopkg.in/yaml.v3"
)

// Request represents HTTP request golden file.
type Request struct {
	*base   `yaml:",inline"`
	Scheme  string   `yaml:"scheme,omitempty"`
	Host    string   `yaml:"host,omitempty"`
	Method  string   `yaml:"method"`
	Path    string   `yaml:"path"`
	Query   string   `yaml:"query"`
	Headers []string `yaml:"headers"`

	Pattern string      `yaml:"-"` // Set based on Method and Path.
	headers http.Header // Parsed Headers field.
	body    Body        // Parsed golden file body field.
	t       tester.T    // Test manager.
}

// NewRequest creates a new [Request] object based on the YAML golden file
// represented by [Source]. The golden file must have a root field named
// "request" which is used to set field values for the [Request] object. On
// error, it marks the test as failed and returns nil.
//
// Based on bodyType the additional "Content-Type" header is always added
// unless it was set explicitly.
//
// Example YAML file:
//
//	request:
//	  scheme: https
//	  host: example.com
//	  method: POST
//	  path: /some/path
//	  query: key0=val0&key1=val1
//	  headers:
//	    - 'Authorization: Bearer token'
//	    - 'Content-Type: application/json'
//	  meta:
//	    key1: val1
//	    key2: 123
//	    key3: 12.3
//	    key4: 2021-02-28T10:24:25.123Z
//	  bodyType: text
//	  body: |
//	    abc
func NewRequest(t tester.T, src Source) *Request {
	t.Helper()

	data, err := io.ReadAll(src)
	if err != nil {
		t.Error(err)
		return nil
	}

	// Request golden file has one root field 'request', so we create
	// a simple wrap and set default field values.
	wrap := struct {
		Request *Request `yaml:"request"`
	}{
		Request: &Request{
			base: &base{
				BodyType: Text,
			},
			Scheme: "http",
			Host:   "localhost",
			t:      t,
		},
	}

	if err = yaml.Unmarshal(data, wrap); err != nil {
		t.Error(err)
		return nil
	}
	req := wrap.Request
	if err = req.setup(src.Path); err != nil {
		t.Error(err)
		return nil
	}
	return req
}

// Request returns [http.Request] represented by the [Request]. On error, it
// will mark the test as failed and return nil.
func (req *Request) Request() *http.Request {
	req.t.Helper()

	uri := &url.URL{
		Scheme:   req.Scheme,
		Host:     req.Host,
		Path:     req.Path,
		RawQuery: req.Query,
	}
	body := bytes.NewReader(req.Body())
	httpReq := httptest.NewRequest(req.Method, uri.String(), body)
	httpReq.URL.RawQuery = req.Query
	httpReq.RequestURI = ""
	httpReq.Header = maps.Clone(req.headers)
	return httpReq
}

// Body returns copy of the request's body as a byte slice.
func (req *Request) Body() []byte {
	return req.body.Body()
}

// Assert asserts request matches the golden file. Returns true on success,
// otherwise it marks the test as failed and returns false.
//
// All headers defined in the golden file must match exactly, but the "have"
// request may have more headers than defined in the golden file.
//
// To compare response bodies, a method best suited for body type is used. For
// example, when comparing JSON bodies, both byte slices don't have to be
// identical, but they must represent the same data.
func (req *Request) Assert(have *http.Request) bool {
	req.t.Helper()

	if req.Scheme != have.URL.Scheme {
		msg := notice.New("expected the request scheme to be equal").
			Want("%s", req.Scheme).
			Have("%s", have.URL.Scheme)
		req.t.Error(msg)
		return false
	}

	if req.Host != have.URL.Host {
		msg := notice.New("expected the request host to be equal").
			Want("%s", req.Host).
			Have("%s", have.URL.Host)
		req.t.Error(msg)
		return false
	}

	if req.Method != have.Method {
		msg := notice.New("expected the request method to be equal").
			Want("%s", req.Method).
			Have("%s", have.Method)
		req.t.Error(msg)
		return false
	}

	if req.Path != have.URL.Path {
		msg := notice.New("expected the request path to be equal").
			Want("%s", req.Path).
			Have("%s", have.URL.Path)
		req.t.Error(msg)
		return false
	}

	if req.Query != have.URL.RawQuery {
		msg := notice.New("expected the request query to be equal").
			Want("%s", req.Query).
			Have("%s", have.URL.RawQuery)
		req.t.Error(msg)
		return false
	}

	if err := check.MapSubset(req.headers, have.Header); err != nil {
		req.t.Error(err)
		return false
	}

	gotBody, rc := cloneReader(req.t, have.Body)
	defer func() { have.Body = rc }()
	return req.body.Assert(req.t, gotBody)
}

// setup takes a path to the YAML golden file and sets up the [Request]. The
// method must be called after unmarshalling the [Request].
func (req *Request) setup(pth string) error {
	var err error

	req.Pattern = req.Method + " " + req.Path

	// Parse request body based on the body type field.
	req.body, err = parseBody(pth, req.RawBody, req.BodyType)
	if err != nil {
		return err
	}

	req.headers, err = lines2Headers(req.Headers...)
	if err != nil {
		return err
	}
	if _, ok := req.headers["Content-Type"]; !ok {
		req.body.SetContentTypeHeader(req.headers)
	} else {
		req.t.Helper()
		msg := "INFO: Content-Type header overwritten by the golden file."
		req.t.Log(msg)
	}
	return req.validate()
}

// validate validates request loaded from golden file.
func (req *Request) validate() error {
	if req.Method == "" {
		return errors.New("HTTP request method field is required")
	}

	if req.Path == "" {
		return errors.New("HTTP request path field is required")
	}
	return nil
}
