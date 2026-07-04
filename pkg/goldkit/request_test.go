package goldkit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/iokit"
)

func Test_NewRequest(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "https", gld.Scheme)
		assert.Equal(t, "example.com", gld.Host)
		assert.Equal(t, http.MethodPost, gld.Method)
		assert.Equal(t, "/some/path", gld.Path)
		assert.Equal(t, "key0=val0&key1=val1", gld.Query)
		assert.Equal(t, "POST /some/path", gld.Pattern)
		wantHeadersSlice := []string{
			"Authorization: Bearer token",
		}
		assert.Equal(t, wantHeadersSlice, gld.Headers)
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"text/plain"},
		}
		assert.MapSubset(t, wantHeadersMap, gld.headers)
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, gld.Meta)
		assert.Equal(t, Text, gld.BodyType)
		assert.Equal(t, "abc\n", string(gld.Body()))
	})

	t.Run("content type header set explicitly", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectLogContain("INFO: Content-Type header overwritten")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_content_type_set.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		wantHeadersSlice := []string{
			"Content-Type: application/json",
		}
		assert.Equal(t, wantHeadersSlice, gld.Headers)
		wantHeadersMap := map[string][]string{
			"Content-Type": {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, gld.headers)
		assert.Equal(t, Text, gld.BodyType)
		assert.Equal(t, "abc", string(gld.Body()))
	})

	t.Run("minimal", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_minimal.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "http", gld.Scheme)
		assert.Equal(t, "localhost", gld.Host)
		assert.Equal(t, http.MethodGet, gld.Method)
		assert.Equal(t, "/", gld.Path)
		assert.Equal(t, "", gld.Query)
		assert.Equal(t, "GET /", gld.Pattern)
		assert.Len(t, 0, gld.Headers)
		wantHeadersMap := http.Header{"Content-Type": {"text/plain"}}
		assert.Equal(t, wantHeadersMap, gld.headers)
		assert.Len(t, 0, gld.Meta)
		assert.Equal(t, Text, gld.BodyType)
		assert.Equal(t, "", string(gld.Body()))
	})

	t.Run("none body", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_none_body.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "http", gld.Scheme)
		assert.Equal(t, "localhost", gld.Host)
		assert.Equal(t, http.MethodGet, gld.Method)
		assert.Equal(t, "/", gld.Path)
		assert.Equal(t, "", gld.Query)
		assert.Equal(t, "GET /", gld.Pattern)
		assert.Len(t, 0, gld.Headers)
		assert.Len(t, 0, gld.headers)
		assert.Len(t, 0, gld.Meta)
		assert.Equal(t, None, gld.BodyType)
		assert.Equal(t, "", string(gld.Body()))
	})

	t.Run("error - none body with body data", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("expected empty body:\n  have:\n        abc\n")
		tspy.Close()

		pth := "testdata/request_none_body_with_body.yml"
		src := must.Value(SourceFrom(pth, nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("multipart", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_multipart.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "http", gld.Scheme)
		assert.Equal(t, "localhost", gld.Host)
		assert.Equal(t, http.MethodPost, gld.Method)
		assert.Equal(t, "/", gld.Path)
		assert.Equal(t, "", gld.Query)
		assert.Len(t, 0, gld.Headers)
		assert.Len(t, 0, gld.Meta)
		assert.Equal(t, Multipart, gld.BodyType)

		body := gld.Body()
		want := "" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"field1\"\r\n" +
			"\r\n" +
			"VALUE1\r\n" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"file1\"; filename=\"file1.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"abc\r\n" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"file2\"; filename=\"file2.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"xyz\r\n" +
			"--{{boundary}}--\r\n"
		want = strings.ReplaceAll(want, "{{boundary}}", must.Value(findBoundary(body)))
		assert.Equal(t, want, string(body))
	})

	t.Run("error reading", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(iokit.ErrRead.Error())
		tspy.Close()

		rdr := iokit.ErrReader(strings.NewReader("abc"), 1)
		src := NewSource("/dir/file", rdr)

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("cannot unmarshal")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/invalid.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("unsupported body type", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(ErrInvBodyType.Error())
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_inv_body_type.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("invalid header", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("malformed MIME header")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_inv_header.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("missing method field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("HTTP request method field is required")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_missing_method.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("missing path field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("HTTP request path field is required")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_missing_path.yml", nil))

		// --- When ---
		gld := NewRequest(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})
}

func Test_Request_Request(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		req := gld.Request()

		// --- Then ---
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https", req.URL.Scheme)
		assert.Equal(t, "example.com", req.Host)
		assert.Equal(t, "/some/path", req.URL.Path)
		assert.Equal(t, "key0=val0&key1=val1", req.URL.RawQuery)
		assert.Len(t, 2, req.Header)
		assert.HasKey(t, "Authorization", req.Header)
		assert.HasKey(t, "Content-Type", req.Header)
		assert.Len(t, 1, req.Header.Values("Authorization"))
		assert.Len(t, 1, req.Header.Values("Content-Type"))
		assert.Equal(t, "Bearer token", req.Header.Values("Authorization")[0])
		assert.Equal(t, "text/plain", req.Header.Values("Content-Type")[0])
	})
}

func Test_Request_Body(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	src := must.Value(SourceFrom("testdata/request_full.yml", nil))

	// --- When ---
	gld := NewRequest(tspy, src)

	// --- Then ---
	assert.Equal(t, "abc\n", string(gld.Body()))
}

func Test_Request_Assert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/some/path",
			strings.NewReader("abc\n"),
		)
		req.Header.Add("Authorization", "Bearer token")
		req.Header.Add("Content-Type", "text/plain")
		req.URL.RawQuery = "key0=val0&key1=val1"

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("scheme does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the request scheme to be equal:\n" +
			"  want: https\n" +
			"  have: http"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"http"+"://example.com/some/path",
			http.NoBody,
		)

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("host does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the request host to be equal:\n" +
			"  want: example.com\n" +
			"  have: other.com"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://other.com/some/path",
			http.NoBody,
		)

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("method does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the request method to be equal:\n" +
			"  want: POST\n" +
			"  have: GET"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodGet,
			"https://example.com/some/path",
			http.NoBody,
		)

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("path does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the request path to be equal:\n" +
			"  want: /some/path\n" +
			"  have: /other/path"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/other/path",
			http.NoBody,
		)

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("query does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the request query to be equal:\n" +
			"  want: key0=val0&key1=val1\n" +
			"  have: key0=val0"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/some/path",
			http.NoBody,
		)
		req.URL.RawQuery = "key0=val0"

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("headers do not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected values to be equal:\n" +
			"  trail: map[\"Authorization\"][0]\n" +
			"   want: \"Bearer token\"\n" +
			"   have: \"Bearer token2\""
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/some/path",
			http.NoBody,
		)
		req.URL.RawQuery = "key0=val0&key1=val1"
		req.Header.Add("Authorization", "Bearer token2")
		req.Header.Add("Content-Type", "text/plain")

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("only defined headers check", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		reqBody := strings.NewReader("abc\n")
		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/some/path",
			reqBody,
		)
		req.URL.RawQuery = "key0=val0&key1=val1"
		req.Header.Add("Authorization", "Bearer token")
		req.Header.Add("Content-Type", "text/plain")
		req.Header.Add("Custom-Header", "custom data")

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("body does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected values to be equal:\n" +
			"  message: TEXT bodies do not match\n" +
			"     want: \"abc\\n\"\n" +
			"     have: \"ABC\\n\"\n" +
			"     diff:\n" +
			"           @@ -1 +1 @@\n" +
			"           -ABC\n" +
			"           +abc"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		req := httptest.NewRequest(
			http.MethodPost,
			"https://example.com/some/path",
			strings.NewReader("ABC\n"),
		)
		req.Header.Add("Authorization", "Bearer token")
		req.Header.Add("Content-Type", "text/plain")
		req.URL.RawQuery = "key0=val0&key1=val1"

		src := must.Value(SourceFrom("testdata/request_full.yml", nil))
		gld := NewRequest(tspy, src)

		// --- When ---
		have := gld.Assert(req)

		// --- Then ---
		assert.False(t, have)
	})
}
