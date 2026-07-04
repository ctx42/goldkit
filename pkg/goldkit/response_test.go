package goldkit

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/iokit"
)

func Test_NewResponse(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Equal(t, 200, gld.StatusCode)
		wantHeadersSlice := []string{
			"Authorization: Bearer token",
			"Content-Type: application/json",
		}
		assert.Equal(t, wantHeadersSlice, gld.Headers)
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, gld.headers)
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, gld.Meta)
		assert.Equal(t, JSON, gld.BodyType)
		assert.JSON(t, `{"key2":"val2"}`, string(gld.Body()))
	})

	t.Run("minimal", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_minimal.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Equal(t, 201, gld.StatusCode)
		assert.Len(t, 0, gld.Headers)
		assert.Len(t, 0, gld.headers)
		assert.Equal(t, "", string(gld.Body()))
		assert.Len(t, 0, gld.Meta)
		assert.Equal(t, Text, gld.BodyType)
	})

	t.Run("multipart", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_multipart.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, 200, gld.StatusCode)
		wantHeadersSlice := []string{
			"X-Custom: value",
		}
		assert.Equal(t, wantHeadersSlice, gld.Headers)
		wantHeadersMap := map[string][]string{
			"X-Custom": {"value"},
		}
		assert.MapSubset(t, wantHeadersMap, gld.headers)
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
		gld := NewResponse(tspy, src)

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
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("unsupported body type", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(ErrInvBodyType.Error())
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_inv_body_type.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("invalid header", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("malformed MIME header")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_inv_header.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("missing status code field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogContain("HTTP response status code field is required")
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_missing_status_code.yml", nil))

		// --- When ---
		gld := NewResponse(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})
}

func Test_Response_Response(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Response()

		// --- Then ---
		assert.Equal(t, 200, have.StatusCode)
		assert.Equal(t, "200 OK", have.Status)
		wantHeadersMap := map[string][]string{
			"Authorization": {"Bearer token"},
			"Content-Type":  {"application/json"},
		}
		assert.MapSubset(t, wantHeadersMap, have.Header)
		assert.JSON(t, `{"key2":"val2"}`, iokit.ReadAllStr(t, have.Body))
		assert.NoError(t, have.Body.Close())
	})

	t.Run("derives Content-Type from body type when not set", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/response_minimal.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Response()

		// --- Then ---
		assert.Equal(t, "text/plain", have.Header.Get("Content-Type"))
	})
}

func Test_Response_Body(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	src := must.Value(SourceFrom("testdata/response_full.yml", nil))

	// --- When ---
	gld := NewResponse(tspy, src)

	// --- Then ---
	assert.Equal(t, `{ "key2": "val2" }`+"\n", string(gld.Body()))
}

func Test_Response_Assert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		rsp := &http.Response{Header: make(http.Header)}
		rsp.StatusCode = 200
		rsp.Header.Add("Authorization", "Bearer token")
		rsp.Header.Add("Content-Type", "application/json")
		rsp.Body = io.NopCloser(strings.NewReader(`{"key2":"val2"}`))

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Assert(rsp)

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("status code does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected response status code to be equal:\n" +
			"  want: 200\n" +
			"  have: 400"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		rsp := &http.Response{Header: make(http.Header)}
		rsp.StatusCode = 400
		rsp.Header.Add("Authorization", "Bearer token")
		rsp.Header.Add("Content-Type", "application/json")
		rsp.Body = io.NopCloser(strings.NewReader(`{"key2":"val2"}`))

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Assert(rsp)

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
			"   have: \"Bearer token 2\""
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		rsp := &http.Response{Header: make(http.Header)}
		rsp.StatusCode = 200
		rsp.Header.Add("Authorization", "Bearer token 2")
		rsp.Header.Add("Content-Type", "application/json")
		rsp.Body = io.NopCloser(strings.NewReader(`{"key2":"val2"}`))

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Assert(rsp)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("only defined headers check", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		rsp := &http.Response{Header: make(http.Header)}
		rsp.StatusCode = 200
		rsp.Header.Add("Authorization", "Bearer token")
		rsp.Header.Add("Content-Type", "application/json")
		rsp.Header.Add("Custom-Header", "custom data")
		rsp.Body = io.NopCloser(strings.NewReader(`{"key2":"val2"}`))

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Assert(rsp)

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("body does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[JSON body] expected JSON strings to be equal:\n" +
			"  want: {\"key2\":\"val2\"}\n" +
			"  have: {\"key2\":\"val1\"}"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		rsp := &http.Response{Header: make(http.Header)}
		rsp.StatusCode = 200
		rsp.Header.Add("Authorization", "Bearer token")
		rsp.Header.Add("Content-Type", "application/json")
		rsp.Header.Add("Custom-Header", "custom data")
		rsp.Body = io.NopCloser(strings.NewReader(`{"key2":"val1"}`))

		src := must.Value(SourceFrom("testdata/response_full.yml", nil))
		gld := NewResponse(tspy, src)

		// --- When ---
		have := gld.Assert(rsp)

		// --- Then ---
		assert.False(t, have)
	})
}
