package goldkit

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testkit/pkg/iokit"
)

func Test_NewMultipart(t *testing.T) {
	// --- When ---
	mp := NewMultipart()

	// --- Then ---
	assert.NotNil(t, mp)
	assert.NotNil(t, mp.w)
	assert.Empty(t, mp.body.String())
}

func Test_MultiPart_SetBoundary(t *testing.T) {
	t.Run("custom boundary", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()

		// --- When ---
		err := mp.SetBoundary("abc")

		// --- Then ---
		assert.NoError(t, err)
		assert.NoError(t, mp.AddField("filed", "value"))
		assert.NoError(t, mp.Close())
		want := "--abc\r\n" +
			"Content-Disposition: form-data; name=\"filed\"\r\n" +
			"\r\n" +
			"value\r\n" +
			"--abc--\r\n"
		assert.Equal(t, want, mp.body.String())
		assert.Equal(t, "abc", mp.Boundary())
	})

	t.Run("invalid boundary", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()

		// --- When ---
		err := mp.SetBoundary("*")

		// --- Then ---
		assert.ErrorEqual(t, "mime: invalid boundary character", err)
	})
}

func Test_MultiPart_AddField(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))

		// --- When ---
		err := mp.AddField("name", "value")

		// --- Then ---
		assert.NoError(t, err)
		assert.NoError(t, mp.Close())
		exp := "--abc\r\n" +
			"Content-Disposition: form-data; name=\"name\"\r\n" +
			"\r\n" +
			"value\r\n--abc--\r\n"
		assert.Equal(t, exp, mp.body.String())
	})

	t.Run("adding when closed causes error", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.Close())

		// --- When ---
		err := mp.AddField("name", "value")

		// --- Then ---
		assert.ErrorIs(t, ErrMultipartClosed, err)
	})
}

func Test_MultiPart_AddFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		rdr := bytes.NewReader([]byte{1, 2, 3})
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))

		// --- When ---
		err := mp.AddFile("name", "filename", rdr)

		// --- Then ---
		assert.NoError(t, err)
		assert.NoError(t, mp.Close())
		exp := "--abc\r\n" +
			"Content-Disposition: form-data; name=\"name\"; filename=\"filename\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"\x01\x02\x03\r\n" +
			"--abc--\r\n"
		assert.Equal(t, exp, mp.body.String())
	})

	t.Run("adding when closed causes error", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.Close())

		// --- When ---
		err := mp.AddFile("name", "filename", nil)

		// --- Then ---
		assert.ErrorIs(t, ErrMultipartClosed, err)
	})

	t.Run("copy error", func(t *testing.T) {
		// --- Given ---
		rdr := iokit.ErrReader(strings.NewReader("abc"), 1)
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))

		// --- When ---
		err := mp.AddFile("name", "filename", rdr)

		// --- Then ---
		assert.ErrorIs(t, iokit.ErrRead, err)
	})
}

func Test_MultiPart_SetContentTypeHeader(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))
		h := make(map[string][]string)

		// --- When ---
		mp.SetContentTypeHeader(h)

		// --- Then ---
		expH := map[string][]string{
			"Content-Type": {"multipart/form-data; boundary=abc"},
		}
		assert.Equal(t, http.Header(expH), http.Header(h))
	})
}

func Test_MultiPart_Request(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		rdr := bytes.NewReader([]byte{1, 2, 3})
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))
		assert.NoError(t, mp.AddField("name", "value"))
		assert.NoError(t, mp.AddFile("name", "filename", rdr))
		assert.NoError(t, mp.Close())

		// --- When ---
		req, err := mp.Request(http.MethodPost, "/abc")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/abc", req.URL.String())
		expH := map[string][]string{
			"Content-Type": {"multipart/form-data; boundary=abc"},
		}
		assert.Equal(t, http.Header(expH), req.Header)

		fil, hdr, err := req.FormFile("name")
		assert.NoError(t, err)
		assert.Equal(t, []byte{1, 2, 3}, iokit.ReadAll(t, fil))
		assert.Equal(t, "filename", hdr.Filename)
		assert.Equal(t, int64(3), hdr.Size)
	})

	t.Run("does not consume the body buffer", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("abc"))
		assert.NoError(t, mp.AddField("name", "value"))

		// --- When ---
		_, err := mp.Request(http.MethodPost, "/abc")

		// --- Then ---
		assert.NoError(t, err)
		want := "--abc\r\n" +
			"Content-Disposition: form-data; name=\"name\"\r\n" +
			"\r\n" +
			"value\r\n" +
			"--abc--\r\n"
		assert.Equal(t, want, string(mp.Body()))
	})

	t.Run("empty body fails to parse", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()

		// --- When ---
		req, err := mp.Request(http.MethodPost, "/abc")

		// --- Then ---
		assert.ErrorEqual(t, "multipart: NextPart: EOF", err)
		assert.Nil(t, req)
	})
}

func Test_MultiPart_Body(t *testing.T) {
	t.Run("writes the trailing boundary", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("boundary"))
		assert.NoError(t, mp.AddField("field", "value"))

		// --- When ---
		have := mp.Body()

		// --- Then ---
		want := "--boundary\r\n" +
			"Content-Disposition: form-data; name=\"field\"\r\n" +
			"\r\n" +
			"value\r\n" +
			"--boundary--\r\n"
		assert.Equal(t, want, string(have))
	})

	t.Run("returns clone", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.SetBoundary("boundary"))
		assert.NoError(t, mp.AddField("field", "value"))

		edited := mp.Body()
		edited[0] = 100

		// --- When ---
		clone := mp.Body()

		// --- Then ---
		assert.NotEqual(t, edited, clone)
	})
}

func Test_MultiPart_Close(t *testing.T) {
	t.Run("closing new", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()

		// --- When ---
		err := mp.Close()

		// --- Then ---
		assert.NoError(t, err)
		assert.Empty(t, mp.body.String())
	})

	t.Run("trite trailing boundary", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.AddField("name", "value"))
		before := mp.body.String()

		// --- When ---
		err := mp.Close()

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, len(before) < mp.body.Len())
	})

	t.Run("double close", func(t *testing.T) {
		// --- Given ---
		mp := NewMultipart()
		assert.NoError(t, mp.AddField("name", "value"))
		assert.NoError(t, mp.Close())

		want := mp.body.String()

		// --- When ---
		err := mp.Close()

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, want, mp.body.String())
	})
}
