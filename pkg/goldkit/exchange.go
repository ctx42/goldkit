package goldkit

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/ctx42/testing/pkg/tester"
	"gopkg.in/yaml.v3"
)

// defExchangeTimeout is the [http.Client] timeout used by [Exchange.Assert]
// when [Exchange.Timeout] is left at its zero value.
const defExchangeTimeout = 30 * time.Second

// Exchange represents HTTP request / response exchange.
type Exchange struct {
	Request  *Request  `yaml:"request"`  // HTTP request.
	Response *Response `yaml:"response"` // HTTP response.

	// Timeout caps the HTTP client used by [Exchange.Assert]; the zero value
	// falls back to defExchangeTimeout so a stalled host cannot hang the test.
	Timeout time.Duration `yaml:"-"`

	t tester.T // Test manager.
}

// NewExchange returns a new instance of HTTP request / response [Exchange].
func NewExchange(t tester.T, src Source) *Exchange {
	t.Helper()
	data, err := io.ReadAll(src)
	if err != nil {
		t.Error(err)
		return nil
	}

	ex := &Exchange{
		Request: &Request{
			base:   &base{BodyType: Text},
			Scheme: "http",
			Host:   "localhost",
			t:      t,
		},
		Response: &Response{base: &base{BodyType: Text}, t: t},
		t:        t,
	}
	if err = yaml.Unmarshal(data, ex); err != nil {
		t.Error(err)
		return nil
	}

	if ex.Request == nil {
		t.Error("HTTP request definition is required")
		return nil
	}
	if err = ex.Request.setup(src.Path); err != nil {
		t.Error(err)
		return nil
	}

	if ex.Response == nil {
		t.Error("HTTP response definition is required")
		return nil
	}
	if err = ex.Response.setup(src.Path); err != nil {
		t.Error(err)
		return nil
	}
	return ex
}

// Assert makes the request described in the golden file to host and asserts
// the response matches. It returns the constructed request and received
// response in case further assertions need to be done.
func (ex *Exchange) Assert() (*http.Request, *http.Response) {
	ex.t.Helper()
	req := ex.Request.Request()

	var reqBody []byte
	reqBody, req.Body = cloneReader(ex.t, req.Body)
	defer func() { req.Body = io.NopCloser(bytes.NewReader(reqBody)) }()

	timeout := ex.Timeout
	if timeout == 0 {
		timeout = defExchangeTimeout
	}
	cli := &http.Client{Timeout: timeout}
	rsp, err := cli.Do(req)
	if err != nil {
		ex.t.Error(err)
		return req, nil
	}
	defer func() { _ = rsp.Body.Close() }()
	ex.Response.Assert(rsp)
	return req, rsp
}

// WriteTo writes golden file to w.
func (ex *Exchange) WriteTo(w io.Writer) (int64, error) {
	data, err := yaml.Marshal(ex)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
