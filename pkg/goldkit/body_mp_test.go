package goldkit

import (
	"io/fs"
	"net/http"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/tester"
	"github.com/ctx42/testkit/pkg/oskit"
	"gopkg.in/yaml.v3"
)

func Test_mpFile_YAML_unmarshal(t *testing.T) {
	// --- Given ---
	data := []byte("field: file0\nname: file0.txt\npath: content0.txt")
	fil := &mpFile{}

	// --- When ---
	err := yaml.Unmarshal(data, fil)

	// --- Then ---
	assert.NoError(t, err)
	assert.Equal(t, "file0", fil.Field)
	assert.Equal(t, "file0.txt", fil.Name)
	assert.Equal(t, "content0.txt", fil.Path)
}

func Test_mpBody_YAML_unmarshal(t *testing.T) {
	// --- Given ---
	data := []byte(`
        files:
          - field: file0
            name: file0.txt
            path: content0.txt
          - field: file1
            name: file1.txt
            path: content1.txt
        values:
          int: [123]
          float: [1.1]
          str: [VALUE1]
          time: [2021-02-03T04:05:06Z]`)
	bdy := &mpBody{}

	// --- When ---
	err := yaml.Unmarshal(data, bdy)

	// --- Then ---
	assert.NoError(t, err)

	wantFiles := []*mpFile{
		{"file0", "file0.txt", "content0.txt"},
		{"file1", "file1.txt", "content1.txt"},
	}
	assert.Equal(t, wantFiles, bdy.Files)

	wantValues := map[string][]string{
		"int":   {"123"},
		"float": {"1.1"},
		"str":   {"VALUE1"},
		"time":  {"2021-02-03T04:05:06Z"},
	}
	assert.Equal(t, wantValues, bdy.Values)
	assert.Equal(t, "", bdy.dir)
	assert.Nil(t, bdy.mp)
}

func Test_newMpBody(t *testing.T) {
	// --- When ---
	bdy := newMpBody("testdata")

	// --- Then ---
	assert.Nil(t, bdy.Files)
	assert.Nil(t, bdy.Values)
	assert.Equal(t, "testdata", bdy.dir)
	assert.NotNil(t, bdy.mp)
}

func Test_mpBody_boundary_setBoundary(t *testing.T) {
	t.Run("by default random string", func(t *testing.T) {
		// --- Given ---
		bdy := newMpBody("testdata")

		// --- When ---
		have := bdy.boundary()

		// --- Then ---
		assert.Len(t, 60, have)
	})

	t.Run("with custom boundary", func(t *testing.T) {
		// --- Given ---
		bdy := newMpBody("testdata")
		assert.NoError(t, bdy.setBoundary("custom-boundary"))

		// --- When ---
		have := bdy.boundary()

		// --- Then ---
		assert.Equal(t, "custom-boundary", have)
	})
}

func Test_mpBody_setBoundary(t *testing.T) {
	t.Run("error - invalid boundary", func(t *testing.T) {
		// --- Given ---
		bdy := newMpBody("testdata")

		// --- When ---
		err := bdy.setBoundary("")

		// --- Then ---
		assert.ErrorEqual(t, "mime: invalid boundary length", err)
	})
}

func Test_mpBody_parse(t *testing.T) {
	t.Run("values only", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        values:
          str: [VALUE1]
          int: [123]
          float: [1.1]
          time: [2021-02-03T04:05:06Z]`)

		bdy := newMpBody("dir")
		assert.NoError(t, bdy.setBoundary("boundary"))
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		assert.NoError(t, err)
		want := "--boundary\r\n" +
			"Content-Disposition: form-data; name=\"float\"\r\n" +
			"\r\n" +
			"1.1\r\n" +
			"--boundary\r\n" +
			"Content-Disposition: form-data; name=\"int\"\r\n" +
			"\r\n" +
			"123\r\n" +
			"--boundary\r\n" +
			"Content-Disposition: form-data; name=\"str\"\r\n" +
			"\r\n" +
			"VALUE1\r\n" +
			"--boundary\r\n" +
			"Content-Disposition: form-data; name=\"time\"\r\n" +
			"\r\n" +
			"2021-02-03T04:05:06Z\r\n" +
			"--boundary--\r\n"
		assert.Equal(t, want, string(bdy.Body()))
	})

	t.Run("file", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        files:
          - field: file1
            name: file1.txt
            path: content0.txt`)

		bdy := newMpBody("testdata")
		assert.NoError(t, bdy.setBoundary("boundary"))
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		assert.NoError(t, err)
		want := "--boundary\r\n" +
			"Content-Disposition: form-data; name=\"file1\"; filename=\"file1.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"abc\r\n" +
			"--boundary--\r\n"
		assert.Equal(t, want, string(bdy.Body()))
	})

	t.Run("filename set when empty", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        files:
          - field: file
            path: bin.wav`)

		bdy := newMpBody("testdata")
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		assert.NoError(t, err)

		files := []*mpFile{{"file", "bin.wav", "testdata/bin.wav"}}
		assert.Equal(t, files, bdy.Files)
	})

	t.Run("binary file", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        files:
          - field: file
            path: bin.wav`)

		bdy := newMpBody("testdata")
		assert.NoError(t, bdy.setBoundary("boundary"))
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		assert.NoError(t, err)
		want := "--boundary\r\n" +
			"Content-Disposition: form-data; name=\"file\"; filename=\"bin.wav\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"RIFF\r\n" +
			"--boundary--\r\n"
		assert.Equal(t, want, string(bdy.Body()))
	})

	t.Run("error - not existing file", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        files:
          - field: file
            path: not-existing.wav`)

		bdy := newMpBody("testdata")
		assert.NoError(t, bdy.setBoundary("boundary"))
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		var e *fs.PathError
		assert.ErrorAs(t, &e, err)
		assert.Equal(t, "testdata/not-existing.wav", e.Path)
		assert.Equal(t, "open", e.Op)
	})

	t.Run("error - add file", func(t *testing.T) {
		// --- Given ---
		data := []byte(`
        files:
          - field: file
            name: dir
            path: .`)

		bdy := newMpBody("testdata")
		assert.NoError(t, bdy.setBoundary("boundary"))
		assert.NoError(t, yaml.Unmarshal(data, bdy))

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		wMsg := "add multipart file \"dir\": read testdata: is a directory"
		assert.ErrorEqual(t, wMsg, err)
	})

	t.Run("error - no files or values", func(t *testing.T) {
		// --- Given ---
		bdy := newMpBody("testdata")
		bdy.Values = map[string][]string{"field": {}}

		// --- When ---
		err := bdy.parse()

		// --- Then ---
		assert.ErrorEqual(t, "multipart body has no files or values", err)
	})
}

func Test_mpBody_Body(t *testing.T) {
	// --- Given ---
	bdy := newMpBody("testdata")
	bdy.Files = []*mpFile{{"file", "filename", "content1.txt"}}
	bdy.Values = map[string][]string{"field1": {"value"}}
	assert.NoError(t, bdy.setBoundary("boundary"))
	assert.NoError(t, bdy.parse())

	// --- When ---
	have := bdy.Body()

	// --- Then ---
	want := "--boundary\r\n" +
		"Content-Disposition: form-data; name=\"field1\"\r\n" +
		"\r\n" +
		"value\r\n" +
		"--boundary\r\n" +
		"Content-Disposition: form-data; name=\"file\"; filename=\"filename\"\r\n" +
		"Content-Type: application/octet-stream\r\n" +
		"\r\n" +
		"xyz\r\n" +
		"--boundary--\r\n"
	assert.Equal(t, want, string(have))
}

func Test_mpBody_Assert(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy1.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("removes multipart temp files", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		dir := t.TempDir()
		tmp := oskit.MkdirAll(t, dir, "tmp")
		t.Setenv("TMPDIR", tmp)

		oskit.Create(t, make([]byte, 11<<20), dir, "big.bin")
		bdy := newMpBody(dir)
		bdy.Files = []*mpFile{{"file", "big.bin", "big.bin"}}
		assert.NoError(t, bdy.parse())

		body := bdy.Body()

		// --- When ---
		have := bdy.Assert(tspy, body)

		// --- Then ---
		assert.True(t, have)
		assert.Len(t, 0, oskit.Readdirnames(t, tmp))
	})

	t.Run("field value does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[MultipartForm.Value] expected values to be equal:\n" +
			"  trail: map[\"field\"][0]\n" +
			"   want: \"value0\"\n" +
			"   have: \"value1\""
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value0"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy1.Values = map[string][]string{"field": {"value1"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("field value count does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[MultipartForm.Value] expected values to be equal:\n" +
			"     trail: map[\"field\"]\n" +
			"  want len: 1\n" +
			"  have len: 2\n" +
			"      want:\n" +
			"            []string{\n" +
			"              \"value0\",\n" +
			"            }\n" +
			"      have:\n            []string{\n" +
			"              \"value0\",\n" +
			"              \"value1\",\n" +
			"            }\n" +
			"      diff:\n" +
			"            @@ -1,4 +1,3 @@\n" +
			"             []string{\n" +
			"            -  \"value0\",\n" +
			"            -  \"value1\",\n" +
			"            +  \"value0\",\n" +
			"             }"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value0"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy1.Values = map[string][]string{"field": {"value0", "value1"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("field count does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[MultipartForm.Value] expected values to be equal:\n" +
			"  want len: 1\n" +
			"  have len: 2\n" +
			"      want:\n" +
			"            map[string][]string{\n" +
			"              \"field\": {\n" +
			"                \"value\",\n" +
			"              },\n" +
			"            }\n" +
			"      have:\n" +
			"            map[string][]string{\n" +
			"              \"field\": {\n" +
			"                \"value\",\n" +
			"              },\n" +
			"              \"other\": []string{\n" +
			"                \"value\",\n" +
			"              },\n" +
			"            }\n" +
			"      diff:\n" +
			"            @@ -1,7 +1,4 @@\n" +
			"             map[string][]string{\n" +
			"            -  \"field\": {\n" +
			"            -    \"value\",\n" +
			"            -  },\n" +
			"            -  \"other\": []string{\n" +
			"            +  \"field\": {\n" +
			"                 \"value\",\n" +
			"               },"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy1.Values = map[string][]string{"field": {"value"}, "other": {"value"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("field missing", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[MultipartForm.Value] expected values to be equal:\n" +
			"  want len: 1\n" +
			"  have len: 0\n" +
			"      want:\n" +
			"            map[string][]string{\n" +
			"              \"field\": {\n" +
			"                \"value\",\n" +
			"              },\n" +
			"            }\n" +
			"      have: map[string][]string{}\n" +
			"      diff:\n" +
			"            @@ -1 +1,5 @@\n" +
			"            -map[string][]string{}\n" +
			"            +map[string][]string{\n" +
			"            +  \"field\": {\n" +
			"            +    \"value\",\n" +
			"            +  },\n" +
			"            +}"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy1.Values = map[string][]string{}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("filename does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the file field to have the filename:\n" +
			"  field: file\n" +
			"   want: filename\n" +
			"   have: other"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"file", "other", "content0.txt"}}
		bdy1.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("the number of files does not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected form to have the same number of files:\n" +
			"  want: 1\n" +
			"  have: 2"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{
			{"file", "filename", "content0.txt"},
			{"file1", "filename1", "content0.txt"},
		}
		bdy1.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("files sharing a field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{
			{"docs", "a.txt", "content0.txt"},
			{"docs", "b.txt", "content1.txt"},
		}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{
			{"docs", "a.txt", "content0.txt"},
			{"docs", "b.txt", "content1.txt"},
		}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		body := bdy1.Body()

		// --- When ---
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("extra file under a shared field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected form to have the same number of files:\n" +
			"  want: 1\n" +
			"  have: 2"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"docs", "a.txt", "content0.txt"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{
			{"docs", "a.txt", "content0.txt"},
			{"docs", "b.txt", "content1.txt"},
		}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		body := bdy1.Body()

		// --- When ---
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("missing file under a shared field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected the file field to have the number of files:\n" +
			"  field: docs\n" +
			"   want: 2\n" +
			"   have: 1"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{
			{"docs", "a.txt", "content0.txt"},
			{"docs", "b.txt", "content1.txt"},
		}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Files = []*mpFile{{"docs", "a.txt", "content0.txt"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		body := bdy1.Body()

		// --- When ---
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("file contents do not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected file content to match:\n" +
			"             field: file\n" +
			"          filename: content0.txt\n" +
			"  first diff index: 1"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "content0.txt", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		// --- When ---
		body := []byte("--boundary0\r\n" +
			"Content-Disposition: form-data; name=\"field\"\r\n" +
			"\r\n" +
			"value\r\n" +
			"--boundary0\r\n" +
			"Content-Disposition: form-data; name=\"file\"; filename=\"content0.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"aBC\r\n" +
			"--boundary0--")
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("file lengths do not match", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"content lengths of files do not match:\n" +
			"  field: file\n" +
			"   file: content0.txt\n" +
			"   want: 3\n" +
			"   have: 4"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "content0.txt", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		// --- When ---
		body := []byte("--boundary0\r\n" +
			"Content-Disposition: form-data; name=\"field\"\r\n" +
			"\r\n" +
			"value\r\n" +
			"--boundary0\r\n" +
			"Content-Disposition: form-data; name=\"file\"; filename=\"content0.txt\"\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"\r\n" +
			"abcd\r\n" +
			"--boundary0--")
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("the boundary is not found in the body", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("find boundary: EOF")
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		// --- When ---
		have := bdy0.Assert(tspy, []byte("garbage"))

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("have body is not valid multipart", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"parse multipart body: " +
			"malformed MIME header: missing colon: \"NotAHeaderLine\""
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		// --- When ---
		body := []byte("--boundary0\r\n" +
			"NotAHeaderLine\r\n" +
			"\r\n" +
			"data\r\n" +
			"--boundary0--\r\n")
		have := bdy0.Assert(tspy, body)

		// --- Then ---
		assert.False(t, have)
	})

	t.Run("have is missing the expected file field", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		tspy.ExpectLogEqual("file field \"file\": http: no such file")
		tspy.Close()

		bdy0 := newMpBody("testdata")
		bdy0.Files = []*mpFile{{"file", "filename", "content0.txt"}}
		bdy0.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy0.setBoundary("boundary0"))
		assert.NoError(t, bdy0.parse())

		bdy1 := newMpBody("testdata")
		bdy1.Values = map[string][]string{"field": {"value"}}
		assert.NoError(t, bdy1.setBoundary("boundary1"))
		assert.NoError(t, bdy1.parse())

		// --- When ---
		have := bdy0.Assert(tspy, bdy1.Body())

		// --- Then ---
		assert.False(t, have)
	})
}

func Test_mpBody_SetContentTypeHeader(t *testing.T) {
	// --- Given ---
	bdy := newMpBody("testdata")
	assert.NoError(t, bdy.setBoundary("abc"))
	h := http.Header{
		"Content-Type": []string{"previous value"},
	}

	// --- When ---
	bdy.SetContentTypeHeader(h)

	// --- Then ---
	want := map[string][]string{
		"Content-Type": {"multipart/form-data; boundary=abc"},
	}
	assert.Equal(t, want, map[string][]string(h))
}
