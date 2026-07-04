package goldkit

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/iokit"
	"github.com/ctx42/testkit/pkg/pathkit"
)

func Test_NewSource(t *testing.T) {
	// --- Given ---
	pth := "/dir/file.txt"
	rdr := strings.NewReader("abc")

	// --- When ---
	src := NewSource(pth, rdr)

	// --- Then ---
	assert.Same(t, rdr, src.Reader)
	assert.Equal(t, pth, src.Path)
}

func Test_SourceFrom(t *testing.T) {
	t.Run("without data", func(t *testing.T) {
		// --- Given ---
		pth := pathkit.AbsPath(t, "testdata/golden.tpl.yml")

		// --- When ---
		src, err := SourceFrom(pth, nil)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, pth, src.Path)
		want := "meta:\n  key1: {{ .key1 }}\n"
		have := iokit.ReadAllStr(t, src)
		assert.Equal(t, want, have)
	})

	t.Run("with data", func(t *testing.T) {
		// --- Given ---
		pth := pathkit.AbsPath(t, "testdata/golden.tpl.yml")
		data := map[string]any{
			"key1": "value1",
		}

		// --- When ---
		src, err := SourceFrom(pth, data)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, pth, src.Path)
		want := "meta:\n  key1: value1\n"
		have := iokit.ReadAllStr(t, src)
		assert.Equal(t, want, have)
	})

	t.Run("custom delim", func(t *testing.T) {
		// --- Given ---
		pth := pathkit.AbsPath(t, "testdata/golden_custom_delim.tpl.yml")
		data := map[string]any{
			"key1": "value1",
		}

		// --- When ---
		src, err := SourceFrom(pth, data, Delims("[[", "]]"))

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, pth, src.Path)
		want := "meta:\n  key1: value1\n"
		have := iokit.ReadAllStr(t, src)
		assert.Equal(t, want, have)
	})

	t.Run("not existing golden file", func(t *testing.T) {
		// --- When ---
		src, err := SourceFrom("testdata/not_existing.yml", nil)

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, pathkit.AbsPath(t, "testdata/not_existing.yml"), e.Path)
		assert.Equal(t, "open", e.Op)
		assert.Zero(t, src)
	})

	t.Run("invalid template", func(t *testing.T) {
		// --- Given ---
		data := map[string]any{"key1": "value1"}

		// --- When ---
		src, err := SourceFrom("testdata/golden_invalid.tpl.yml", data)

		// --- Then ---
		wMsg := "template: golden:2: unexpected \"}\" in operand"
		assert.ErrorEqual(t, wMsg, err)
		assert.Zero(t, src)
	})

	t.Run("invalid template data", func(t *testing.T) {
		// --- Given ---
		data := map[string]any{"key1": func() {}}

		// --- When ---
		src, err := SourceFrom("testdata/golden.tpl.yml", data)

		// --- Then ---
		wMsg := "template: " +
			"golden:2:11: " +
			"executing \"golden\" at <{{.key1}}>: " +
			"can't print {{.key1}} of type func()"
		assert.ErrorEqual(t, wMsg, err)
		assert.Zero(t, src)
	})
}

func Test_Open(t *testing.T) {
	t.Run("without data", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		pth := pathkit.AbsPath(t, "testdata/golden.tpl.yml")

		// --- When ---
		tm, src := Open(tspy, pth, nil)

		// --- Then ---
		tspy.Finish()

		assert.Same(t, tspy, tm)
		assert.Equal(t, pth, src.Path)
		want := "meta:\n  key1: {{ .key1 }}\n"
		have := iokit.ReadAllStr(t, src)
		assert.Equal(t, want, have)
	})

	t.Run("with data", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		pth := pathkit.AbsPath(t, "testdata/golden.tpl.yml")
		data := map[string]any{
			"key1": "value1",
		}

		// --- When ---
		tm, src := Open(tspy, pth, data)

		// --- Then ---
		tspy.Finish()

		assert.Same(t, tspy, tm)
		assert.Equal(t, pth, src.Path)
		want := "meta:\n  key1: value1\n"
		have := iokit.ReadAllStr(t, src)
		assert.Equal(t, want, have)
	})

	t.Run("custom delim", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		pth := pathkit.AbsPath(t, "testdata/golden_custom_delim.tpl.yml")
		data := map[string]any{
			"key1": "value1",
		}

		// --- When ---
		tm, src := Open(tspy, pth, data, Delims("[[", "]]"))

		// --- Then ---
		tspy.Finish()
		assert.Same(t, tspy, tm)
		assert.Equal(t, pth, src.Path)
		have := iokit.ReadAllStr(t, src)
		want := "meta:\n  key1: value1\n"
		assert.Equal(t, want, have)
	})

	t.Run("invalid template", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "template: golden:2: unexpected \"}\" in operand"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		data := map[string]any{
			"key1": "value1",
		}

		// --- When ---
		tm, src := Open(tspy, "testdata/golden_invalid.tpl.yml", data)

		// --- Then ---
		tspy.Finish()
		assert.Same(t, tspy, tm)
		assert.Zero(t, src)
	})
}
