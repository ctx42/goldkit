package goldkit

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/iokit"
	"github.com/ctx42/testkit/pkg/oskit"
)

func Test_New(t *testing.T) {
	t.Run("without metadata", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/file.yml", nil))

		// --- When ---
		gld := New(tspy, src)

		// --- Then ---
		assert.Equal(t, JSON, gld.BodyType)
		assert.JSON(t, `{"key1": "val1"}`, string(gld.Body()))
	})

	t.Run("with metadata", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		data := map[string]any{"data": "data line"}
		src := must.Value(SourceFrom("testdata/file_metadata.yml", data))

		// --- When ---
		gld := New(tspy, src)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "Line1\nLine2\ndata line", string(gld.Body()))
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, gld.Meta)
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
		gld := New(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("invalid YAML file", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "yaml: unmarshal errors:\n" +
			"  line 1: cannot unmarshal !!! `` into goldkit.File"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/invalid.yml", nil))

		// --- When ---
		gld := New(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})

	t.Run("unsupported body type", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual(ErrInvBodyType.Error())
		tspy.Close()

		src := must.Value(SourceFrom("testdata/file_inv_body_type.yml", nil))

		// --- When ---
		gld := New(tspy, src)

		// --- Then ---
		assert.Nil(t, gld)
	})
}

func Test_Create(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		// --- When ---
		gld := Create(tspy, "testdata/file.yml", nil)

		// --- Then ---
		assert.Equal(t, JSON, gld.BodyType)
		assert.Empty(t, gld.Meta)
		assert.Equal(t, `{ "key1": "val1" }`, string(gld.Body()))
	})

	t.Run("templated", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		data := map[string]any{"val": "my-value"}

		// --- When ---
		gld := Create(tspy, "testdata/file.tpl.yml", data)

		// --- Then ---
		assert.Equal(t, JSON, gld.BodyType)
		assert.Empty(t, gld.Meta)
		assert.Equal(t, `{ "key1": "my-value" }`, string(gld.Body()))
	})

	t.Run("with metadata", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		data := map[string]any{"data": "my-line"}

		// --- When ---
		gld := Create(tspy, "testdata/file_metadata.yml", data)

		// --- Then ---
		assert.NotNil(t, gld)
		assert.Equal(t, "Line1\nLine2\nmy-line", string(gld.Body()))
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
		}
		assert.Equal(t, wantMeta, gld.Meta)
	})
}

func Test_File_Body(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	src := must.Value(SourceFrom("testdata/file.yml", nil))
	gld := New(tspy, src)

	// --- When ---
	have := gld.Body()

	// --- Then ---
	want := []byte(`{ "key1": "val1" }`)
	assert.Equal(t, want, have)
}

func Test_File_Reader(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	src := must.Value(SourceFrom("testdata/file.yml", nil))
	gld := New(tspy, src)

	// --- When ---
	rdr := gld.Reader()

	// --- Then ---
	want := []byte(`{ "key1": "val1" }`)
	have := iokit.ReadAll(t, rdr)
	assert.Equal(t, want, have)
}

func Test_File_Assert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/file.yml", nil))
		gld := New(tspy, src)

		// --- When ---
		have := gld.Assert([]byte(`{"key1":"val1"}`))

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("failure", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[JSON body] expected JSON strings to be equal:\n" +
			"  want: {\"key1\":\"val1\"}\n" +
			"  have: {\"key1\":\"val0\"}"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		src := must.Value(SourceFrom("testdata/file.yml", nil))
		gld := New(tspy, src)

		// --- When ---
		have := gld.Assert([]byte(`{"key1":"val0"}`))

		// --- Then ---
		assert.False(t, have)
	})
}

func Test_File_WriteTo(t *testing.T) {
	// --- Given ---
	tspy := tester.New(t)
	tspy.Close()

	src := must.Value(SourceFrom("testdata/file_metadata.yml", nil))
	gld := New(tspy, src)
	dst := &bytes.Buffer{}

	// --- When ---
	n, err := gld.WriteTo(dst)

	// --- Then ---
	assert.NoError(t, err)
	assert.Equal(t, int64(141), n)
	want := oskit.ReadFileStr(t, "testdata", "file_write_to.yml")
	assert.Equal(t, want, dst.String())
}
