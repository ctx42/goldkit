package goldkit

import (
	"bytes"
	"io"

	"github.com/ctx42/testing/pkg/tester"
	"gopkg.in/yaml.v3"
)

// File represents a golden file.
type File struct {
	*base `yaml:",inline"`
	body  Body     // File body.
	t     tester.T // Test manager.
}

// New returns golden [File] representation.
func New(t tester.T, src Source) *File {
	t.Helper()

	data, err := io.ReadAll(src)
	if err != nil {
		t.Error(err)
		return nil
	}

	fil := &File{
		base: &base{
			BodyType: Text,
		},
		t: t,
	}
	if err = yaml.Unmarshal(data, fil); err != nil {
		t.Error(err)
		return nil
	}

	if err = fil.setup(src.Path); err != nil {
		t.Error(err)
		return nil
	}
	return fil
}

// Create is a convenience function for:
//
//	gld := goldkit.Create(t, "golden.yml", data)
func Create(t tester.T, pth string, data any, opts ...Option) *File {
	return New(Open(t, pth, data, opts...))
}

// Body returns the file's body as a byte slice.
func (fil *File) Body() []byte {
	return fil.body.Body()
}

// Reader returns reader for body.
func (fil *File) Reader() io.Reader {
	return bytes.NewReader(fil.body.Body())
}

// Assert asserts file body matches 'have'. It chooses the best way to compare
// two byte slices based on body type. For example, when comparing JSON both
// byte slices don't have to be identical, but they must represent the same
// data.
func (fil *File) Assert(have []byte) bool {
	fil.t.Helper()
	return fil.body.Assert(fil.t, have)
}

// WriteTo writes golden file to w.
func (fil *File) WriteTo(w io.Writer) (int64, error) {
	data, err := yaml.Marshal(fil)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}

// setup takes the path to the golden file and sets up the [File]. The method
// must be called after unmarshalling the File.
func (fil *File) setup(pth string) error {
	var err error

	// Parse the body based on the body type field.
	fil.body, err = parseBody(pth, fil.RawBody, fil.BodyType)
	return err
}
