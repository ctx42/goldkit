package goldkit

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/httpkit"
	"github.com/ctx42/testkit/pkg/iokit"
	"github.com/ctx42/testkit/pkg/netkit"
	"github.com/ctx42/testkit/pkg/oskit"
)

func Test_NewExchange(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		data := Meta{}.MetaSet("host", "example.com")
		src := must.Value(SourceFrom("testdata/exchange.yml", data))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld.Request)
		assert.NotNil(t, gld.Response)

		// Request
		req := gld.Request
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "http", req.Scheme)
		assert.Equal(t, "example.com", req.Host)
		assert.Equal(t, "/some/path", req.Path)
		assert.Equal(t, "key0=val0&key1=val1", req.Query)
		wantHeadersSlice := []string{
			"Authorization: Bearer token",
		}
		assert.Equal(t, wantHeadersSlice, req.Headers)
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, req.headers)
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, req.Meta)
		assert.Equal(t, JSON, req.BodyType)
		assert.JSON(t, `{"key2": "val2"}`, string(req.Body()))

		// Response
		rsp := gld.Response
		assert.Equal(t, 200, rsp.StatusCode)
		wantHeadersSlice = []string{
			"Content-Type: application/json",
		}
		assert.Equal(t, wantHeadersSlice, rsp.Headers)
		wantHeadersMap = map[string][]string{
			"Content-Type": {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, rsp.headers)
		wantMeta = map[string]any{
			"key1": "val2",
			"key2": 456,
			"key3": 4.56,
			"key4": time.Date(2001, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, rsp.Meta)
		assert.Equal(t, JSON, rsp.BodyType)
		assert.JSON(t, `{"success": true}`, string(rsp.Body()))
	})

	t.Run("request scheme and host default", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/exchange_minimal.yml", nil))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "http", gld.Request.Scheme)
		assert.Equal(t, "localhost", gld.Request.Host)
	})

	t.Run("read error", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(iokit.ErrRead.Error())
		tspy.Close()

		rdr := iokit.ErrReader(strings.NewReader("abc"), 1)
		src := NewSource("/dir/file", rdr)

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("unmarshall error", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "yaml: unmarshal errors:\n" +
			"  line 1: cannot unmarshal !!! `` into goldkit.Exchange"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		src := NewSource("/dir/file.txt", strings.NewReader("!!!"))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("request unknown body type", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(ErrInvBodyType.Error())
		tspy.Close()

		pth := "testdata/exchange_req_inv_body_type.yml"
		src := must.Value(SourceFrom(pth, nil))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("response no status", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("HTTP response status code field is required")
		tspy.Close()

		pth := "testdata/exchange_rsp_missing_status.yml"
		src := must.Value(SourceFrom(pth, nil))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("error - null request", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("HTTP request definition is required")
		tspy.Close()

		pth := "testdata/exchange_null_request.yml"
		src := must.Value(SourceFrom(pth, nil))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("error - null response", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("HTTP response definition is required")
		tspy.Close()

		pth := "testdata/exchange_null_response.yml"
		src := must.Value(SourceFrom(pth, nil))

		// --- When ---
		gld := NewExchange(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})
}

func Test_Exchange_Assert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		srv := httpkit.NewServer(t)
		srv.Rsp(http.StatusOK, []byte(`{"success": true}`)).
			Header("Content-Type", "application/json")

		u := must.Value(url.Parse(srv.URL()))
		data := Meta{}.MetaSet("host", u.Host)
		src := must.Value(SourceFrom("testdata/exchange.yml", data))
		gld := NewExchange(tspy, src)

		// --- When ---
		req, res := gld.Assert()

		// --- Then ---
		assert.Equal(t, http.MethodPost, req.Method)
		wantURL := "http://127.0.0.1:%s/some/path?key0=val0&key1=val1"
		wantURL = fmt.Sprintf(wantURL, srv.Port())
		assert.Equal(t, wantURL, req.URL.String())
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, req.Header)
		assert.JSON(t, `{"key2": "val2"}`, iokit.ReadAllStr(t, req.Body))
		assert.Equal(t, srv.Host(), req.Host)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		wantHeadersMap = map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {"17"},
		}
		assert.MapSubset(t, wantHeadersMap, res.Header)
		assert.JSON(t, `{"success": true}`, iokit.ReadAllStr(t, res.Body))
		assert.NoError(t, res.Body.Close())
	})

	t.Run("response body does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[JSON body] expected JSON strings to be equal:\n" +
			"  want: {\"success\":true}\n" +
			"  have: {\"success\":false}"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		srv := httpkit.NewServer(t)
		srv.Rsp(http.StatusOK, []byte(`{"success": false}`)).
			Header("Content-Type", "application/json")

		u := must.Value(url.Parse(srv.URL()))
		data := Meta{}.MetaSet("host", u.Host)
		src := must.Value(SourceFrom("testdata/exchange.yml", data))
		gld := NewExchange(tspy, src)

		// --- When ---
		req, res := gld.Assert()

		// --- Then ---
		assert.Equal(t, http.MethodPost, req.Method)
		wantURL := "http://127.0.0.1:%s/some/path?key0=val0&key1=val1"
		wantURL = fmt.Sprintf(wantURL, srv.Port())
		assert.Equal(t, wantURL, req.URL.String())
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, req.Header)
		assert.JSON(t, `{"key2": "val2"}`, iokit.ReadAllStr(t, req.Body))
		assert.Equal(t, srv.Host(), req.Host)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		wantHeadersMap = map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {"18"},
		}
		assert.MapSubset(t, wantHeadersMap, res.Header)
		assert.JSON(t, `{"success": false}`, iokit.ReadAllStr(t, res.Body))
		assert.NoError(t, res.Body.Close())
	})

	t.Run("respects the client timeout", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("Client.Timeout exceeded while awaiting headers")
		tspy.Close()

		block := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) { <-block },
		))
		t.Cleanup(func() { close(block); srv.Close() })

		u := must.Value(url.Parse(srv.URL))
		data := Meta{}.MetaSet("host", u.Host)
		src := must.Value(SourceFrom("testdata/exchange.yml", data))
		gld := NewExchange(tspy, src)
		gld.Timeout = 20 * time.Millisecond

		// --- When ---
		req, res := gld.Assert() //nolint:bodyclose

		// --- Then ---
		assert.NotNil(t, req)
		assert.Nil(t, res)
	})

	t.Run("connection refused", func(t *testing.T) {
		// --- Given ---
		port := must.Value(netkit.GetFreePort())
		assert.NoError(t, netkit.ReservePort(port))

		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "Post \"http://127.0.0.1:%d/some/path?key0=val0&key1=val1\": " +
			"dial tcp 127.0.0.1:%d: connect: connection refused"
		tspy.ExpectLogEqual(wMsg, port, port)
		tspy.Close()

		data := Meta{}.MetaSet("host", "127.0.0.1:"+strconv.Itoa(port))
		src := must.Value(SourceFrom("testdata/exchange.yml", data))
		gld := NewExchange(tspy, src)

		// --- When ---
		req, res := gld.Assert() //nolint:bodyclose

		// --- Then ---
		assert.Equal(t, http.MethodPost, req.Method)
		wantURL := "http://127.0.0.1:%d/some/path?key0=val0&key1=val1"
		wantURL = fmt.Sprintf(wantURL, port)
		assert.Equal(t, wantURL, req.URL.String())
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, req.Header)
		assert.JSON(t, `{"key2": "val2"}`, iokit.ReadAllStr(t, req.Body))
		assert.Equal(t, "127.0.0.1:"+strconv.Itoa(port), req.Host)

		assert.Nil(t, res)
	})
}

func Test_Exchange_WriteTo(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	data := Meta{}.MetaSet("host", "example.com")
	src := must.Value(SourceFrom("testdata/exchange.yml", data))

	gld := NewExchange(tspy, src)
	dst := &bytes.Buffer{}

	// --- When ---
	n, err := gld.WriteTo(dst)

	// --- Then ---
	assert.NoError(t, err)
	assert.Equal(t, int64(593), n)
	want := oskit.ReadFileStr(t, "testdata", "exchange_write_to.yml")
	assert.Equal(t, want, dst.String())
}
