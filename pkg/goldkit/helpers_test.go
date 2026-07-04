package goldkit

import (
	"io"
	"strings"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/iokit"
	"gopkg.in/yaml.v3"
)

func Test_parseBody(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		// --- Given ---
		node := yaml.Node{
			Kind:   yaml.ScalarNode,
			Style:  yaml.LiteralStyle,
			Tag:    "!!str",
			Value:  `{}`,
			Line:   1,
			Column: 2,
		}

		// --- When ---
		bdy, err := parseBody("/", node, JSON)

		// --- Then ---
		assert.NoError(t, err)

		assert.SameType(t, &bodyJSON{}, bdy)
		assert.SameType(t, `{}`, string(bdy.Body()))
	})

	t.Run("text", func(t *testing.T) {
		// --- Given ---
		node := yaml.Node{
			Kind:   yaml.ScalarNode,
			Style:  yaml.LiteralStyle,
			Tag:    "!!str",
			Value:  "abc\n",
			Line:   1,
			Column: 2,
		}

		// --- When ---
		bdy, err := parseBody("/", node, Text)

		// --- Then ---
		assert.NoError(t, err)

		assert.SameType(t, &bodyText{}, bdy)
		assert.Equal(t, "abc\n", string(bdy.Body()))
	})

	t.Run("none", func(t *testing.T) {
		// --- Given ---
		node := yaml.Node{
			Kind:   yaml.ScalarNode,
			Style:  yaml.LiteralStyle,
			Tag:    "!!str",
			Value:  "",
			Line:   1,
			Column: 2,
		}

		// --- When ---
		bdy, err := parseBody("/", node, None)

		// --- Then ---
		assert.NoError(t, err)

		assert.SameType(t, bodyNone{}, bdy)
		assert.Nil(t, bdy.Body())
	})

	t.Run("multi part", func(t *testing.T) {
		// --- Given ---
		content := []byte(`
body:
  files:
    - field: file0
      name: file0.txt
      path: content0.txt
    - field: file1
      name: file1.txt
      path: content1.txt
  values:
    int: [123]
    str: [VALUE1]`)
		var node base
		assert.NoError(t, yaml.Unmarshal(content, &node))

		// --- When ---
		bdy, err := parseBody("testdata/golden.yml", node.RawBody, Multipart)

		// --- Then ---
		assert.NoError(t, err)

		assert.SameType(t, &mpBody{}, bdy)
		body := bdy.Body()
		want := "" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"int\"\r\n" +
			"\r\n" +
			"123\r\n" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"str\"\r\n" +
			"\r\n" +
			"VALUE1\r\n" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"file0\"; filename=\"file0.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"abc\r\n" +
			"--{{boundary}}\r\n" +
			"Content-Disposition: form-data; name=\"file1\"; filename=\"file1.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"xyz\r\n" +
			"--{{boundary}}--\r\n" +
			""
		want = strings.ReplaceAll(
			want,
			"{{boundary}}",
			must.Value(findBoundary(body)),
		)
		assert.Equal(t, want, string(body))
	})

	t.Run("multipart parse error", func(t *testing.T) {
		// --- Given ---
		content := []byte(`
body:
  files:
    - field: file0
      name: file0.txt
      path: does-not-exist.txt`)
		var node base
		assert.NoError(t, yaml.Unmarshal(content, &node))

		// --- When ---
		bdy, err := parseBody("testdata/golden.yml", node.RawBody, Multipart)

		// --- Then ---
		assert.ErrorContain(t, "does-not-exist.txt", err)
		assert.Nil(t, bdy)
	})

	t.Run("multipart decode error", func(t *testing.T) {
		// --- Given ---
		node := yaml.Node{
			Kind:   yaml.ScalarNode,
			Style:  yaml.LiteralStyle,
			Tag:    "!!str",
			Value:  "",
			Line:   1,
			Column: 2,
		}

		// --- When ---
		bdy, err := parseBody("/", node, Multipart)

		// --- Then ---
		assert.ErrorContain(t, "yaml: unmarshal errors", err)
		assert.Nil(t, bdy)
	})

	t.Run("unknown", func(t *testing.T) {
		// --- Given ---
		node := yaml.Node{
			Kind:   yaml.ScalarNode,
			Style:  yaml.LiteralStyle,
			Tag:    "!!str",
			Value:  "",
			Line:   1,
			Column: 2,
		}

		// --- When ---
		bdy, err := parseBody("/", node, "unknown")

		// --- Then ---
		assert.ErrorIs(t, ErrInvBodyType, err)
		assert.Nil(t, bdy)
	})
}

func Test_cloneReader(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		rdr := io.NopCloser(strings.NewReader("some text"))

		// --- When ---
		buf, got := cloneReader(tspy, rdr)

		// --- Then ---
		assert.Equal(t, "some text", string(buf))
		assert.Equal(t, "some text", iokit.ReadAllStr(t, got))
	})

	t.Run("reader error", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(iokit.ErrRead.Error())
		tspy.Close()

		r := iokit.ErrReader(strings.NewReader("some text"), 2)
		rdr := io.NopCloser(r)

		// --- When ---
		buf, got := cloneReader(tspy, rdr)

		// --- Then ---
		assert.Nil(t, buf)
		assert.Nil(t, got)
	})
}

func Test_lines2Headers(t *testing.T) {
	t.Run("multiple headers", func(t *testing.T) {
		// --- Given ---
		lns := []string{
			"Authorization: Bearer token",
			"Custom-Header: val0",
			"Custom-Header: val1",
		}

		// --- When ---
		hs, err := lines2Headers(lns...)

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 2, hs)
		assert.HasKey(t, "Authorization", hs)
		assert.HasKey(t, "Custom-Header", hs)
		assert.Len(t, 1, hs.Values("Authorization"))
		assert.Len(t, 2, hs.Values("Custom-Header"))
		assert.Equal(t, "Bearer token", hs.Get("Authorization"))
		assert.Equal(t, "val0", hs.Values("Custom-Header")[0])
		assert.Equal(t, "val1", hs.Values("Custom-Header")[1])
	})

	t.Run("no lines empty header", func(t *testing.T) {
		// --- When ---
		hs, err := lines2Headers()

		// --- Then ---
		assert.NoError(t, err)
		assert.Len(t, 0, hs)
	})

	t.Run("error", func(t *testing.T) {
		// --- When ---
		hs, err := lines2Headers("abc")

		// --- Then ---
		wMsg := "malformed MIME header: missing colon: \"abc\""
		assert.ErrorContain(t, wMsg, err)
		assert.Len(t, 0, hs)
	})
}

func Test_findBoundary(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		// --- Given ---
		body := []byte("--abc\r\n")

		// --- When ---
		boundary, err := findBoundary(body)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "abc", boundary)
	})

	t.Run("boundary starting with a dash", func(t *testing.T) {
		// --- Given ---
		body := []byte("---xyz\r\n")

		// --- When ---
		boundary, err := findBoundary(body)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "-xyz", boundary)
	})

	t.Run("invalid", func(t *testing.T) {
		// --- Given ---
		body := []byte("abc\n")

		// --- When ---
		boundary, err := findBoundary(body)

		// --- Then ---
		assert.ErrorEqual(t, "find boundary: invalid multipart body", err)
		assert.Equal(t, "", boundary)
	})

	t.Run("error reading", func(t *testing.T) {
		// --- Given ---
		body := []byte("")

		// --- When ---
		boundary, err := findBoundary(body)

		// --- Then ---
		assert.ErrorIs(t, io.EOF, err)
		assert.ErrorContain(t, "find boundary", err)
		assert.Equal(t, "", boundary)
	})
}
