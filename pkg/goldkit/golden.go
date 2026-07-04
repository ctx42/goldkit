// Package goldkit provides structures helping to test with golden files
// defined as YAML. Golden files may hold plain, templated, or JSON content, or
// describe full HTTP request/response exchanges, including multipart bodies.
package goldkit

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ctx42/testing/pkg/tester"
)

// Golden file body types.
const (
	// Text represents text body type (default).
	Text = "text"

	// JSON represents JSON body type.
	JSON = "json"

	// Multipart represents the HTTP multipart body type.
	Multipart = "multipart"

	// None represents not existing body.
	None = "none"
)

// Body is an interface representing golden file body.
type Body interface {
	// Body returns body as a byte slice. Every call should return a new slice.
	Body() []byte

	// Assert returns true when body equals to `have`, false otherwise.
	Assert(t tester.T, have []byte) bool

	// SetContentTypeHeader sets Content-Type header in the given map.
	SetContentTypeHeader(h http.Header)
}

// Option represents a template parsing option.
type Option func(tpl *template.Template) *template.Template

// Delims sets the template action delimiters to the specified strings.
func Delims(left, right string) Option {
	return func(tpl *template.Template) *template.Template {
		return tpl.Delims(left, right)
	}
}

// Source represents golden file source.
type Source struct {
	io.Reader        // Golden file source reader.
	Path      string // Path to the golden file.
}

// NewSource returns new instance of Source.
func NewSource(pth string, rdr io.Reader) Source {
	return Source{
		Reader: rdr,
		Path:   pth,
	}
}

// SourceFrom returns a new [Source] instance based on the given file path,
// data, and options. The function reads the contents of the file at the
// provided path and processes it as a Go template if the data parameter is not
// nil. If the data parameter is not nil, the function will use the template
// package to parse the file's content, replace the placeholders in the file
// with values from the data parameter, and set the Source's Reader field to a
// buffer containing the processed template, and the Path field to the provided
// path.
//
// The function also accepts an optional list of [Option] functions that can
// modify the behavior of the template. If any errors occur during the process,
// the function will return an empty [Source] and the corresponding error.
//
// Example usage:
//
//	src, err := SourceFrom(pth, nil)
//	src, err := SourceFrom(pth, data)
//	src, err := SourceFrom(pth, data, Delims("[[", "]]"))
func SourceFrom(pth string, data any, opts ...Option) (Source, error) {
	var err error
	if pth, err = filepath.Abs(pth); err != nil {
		return Source{}, err
	}

	content, err := os.ReadFile(pth) //nolint:gosec // caller-supplied path.
	if err != nil {
		return Source{}, err
	}

	if data == nil {
		return Source{Reader: bytes.NewReader(content), Path: pth}, nil
	}

	tpl := template.New("golden")
	for _, opt := range opts {
		tpl = opt(tpl)
	}
	if tpl, err = tpl.Parse(string(content)); err != nil {
		return Source{}, err
	}

	buf := &bytes.Buffer{}
	if err = tpl.Execute(buf, data); err != nil {
		return Source{}, err
	}
	return Source{Reader: buf, Path: pth}, nil
}

// Open reads a golden file pointed by the path "pth". Returns the same test
// manager which was passed to it and a [Source].
//
// If data is not nil, the golden file pointed by pth is treated as a template
// and applies a parsed template to the specified data object.
//
// You can set template action delimiters using [Delims] function:
//
//	Open(t, "golden.yml", data, Delims("[[", "]]"))
func Open(t tester.T, pth string, data any, opts ...Option) (tester.T, Source) {
	t.Helper()
	src, err := SourceFrom(pth, data, opts...)
	if err != nil {
		t.Error(err)
		return t, Source{}
	}
	return t, src
}
