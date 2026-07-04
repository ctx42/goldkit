package goldkit

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"

	"github.com/ctx42/testing/pkg/check"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
)

// mpFile describes a file which is part of HTTP multipart request / response.
type mpFile struct {
	Field string `yaml:"field"` // Field name.
	Name  string `yaml:"name"`  // Filename.
	Path  string `yaml:"path"`  // Absolute or golden-relative file path.
}

// mpBody represents HTTP multipart body.
// It's unmarshalled from YAML. Example:
//
//	files:                  # Array of files (optional).
//	  - field: file1        # Form field name.
//	    name: file1.txt     # Filename; defaults to path base (optional).
//	    path: content0.txt  # File path, absolute or golden-relative.
//	values:                 # Map of values (optional).
//	  field1: VALUE1
type mpBody struct {
	Files  []*mpFile           `yaml:"files"`  // Zero or more files.
	Values map[string][]string `yaml:"values"` // Zero or more values.
	dir    string              // Path to the golden file directory.
	mp     *MultiPart          // Multipart body builder.
}

// newMpBody takes path to golden file directory and returns new instance
// of mpBody.
func newMpBody(dir string) *mpBody {
	return &mpBody{
		dir: dir,
		mp:  NewMultipart(),
	}
}

// boundary returns the multipart boundary separator.
func (bdy *mpBody) boundary() string {
	return bdy.mp.Boundary()
}

// setBoundary sets multipart boundary separator.
func (bdy *mpBody) setBoundary(boundary string) error {
	return bdy.mp.SetBoundary(boundary)
}

// parse creates a multipart body based on [mpBody.Files] and [mpBody.Values].
func (bdy *mpBody) parse() error {
	var keys []string
	for key := range bdy.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, val := range bdy.Values[key] {
			if err := bdy.mp.AddField(key, val); err != nil {
				return fmt.Errorf("add multipart field %q: %w", key, err)
			}
		}
	}
	for _, fil := range bdy.Files {
		if !filepath.IsAbs(fil.Path) {
			fil.Path = filepath.Join(bdy.dir, fil.Path)
		}
		f, err := os.Open(fil.Path)
		if err != nil {
			return fmt.Errorf("open multipart file: %w", err)
		}
		if fil.Name == "" {
			fil.Name = filepath.Base(fil.Path)
		}
		if err = bdy.mp.AddFile(fil.Field, fil.Name, f); err != nil {
			_ = f.Close()
			return fmt.Errorf("add multipart file %q: %w", fil.Name, err)
		}
		_ = f.Close()
	}
	if err := bdy.mp.Close(); err != nil {
		return fmt.Errorf("close multipart body: %w", err)
	}
	return nil
}

func (bdy *mpBody) Body() []byte {
	return bdy.mp.Body()
}

func (bdy *mpBody) Assert(t tester.T, have []byte) bool {
	t.Helper()
	if err := bdy.assert(have); err != nil {
		t.Error(err)
		return false
	}
	return true
}

// assert asserts multipart bodies are the same.
//
//nolint:gocognit,cyclop
func (bdy *mpBody) assert(have []byte) error {
	wantReq, err := bdy.mp.Request(http.MethodPost, "/")
	if err != nil {
		return err
	}
	wantValMap := wantReq.MultipartForm.Value
	wantFiles := wantReq.MultipartForm.File

	boundary, err := findBoundary(have)
	if err != nil {
		return err
	}
	haveReq := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(have))
	haveReq.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	if err = haveReq.ParseMultipartForm(10e6); err != nil {
		return err
	}
	haveValMap := haveReq.MultipartForm.Value
	haveFiles := haveReq.MultipartForm.File

	if err = check.Equal(wantValMap, haveValMap); err != nil {
		return notice.From(err, "MultipartForm.Value")
	}

	for _, fil := range bdy.Files {
		_, mh, err := haveReq.FormFile(fil.Field)
		if err != nil {
			return err
		}
		if fil.Name != mh.Filename {
			return notice.New("expected the file field to have the filename").
				Append("field", "%s", fil.Field).
				Append("want", "%s", fil.Name).
				Append("have", "%s", mh.Filename)
		}

		wantContent, err := os.ReadFile(fil.Path)
		if err != nil {
			return err
		}

		haveFil, err := mh.Open()
		if err != nil {
			return err
		}
		haveContent, err := io.ReadAll(haveFil)
		if err != nil {
			return err
		}
		_ = haveFil.Close()

		wantLen := len(wantContent)
		haveLen := len(haveContent)
		if wantLen != haveLen {
			return notice.New("content lengths of files do not match").
				Append("field", "%s", fil.Field).
				Append("file", "%s", fil.Name).
				Want("%d", wantLen).
				Have("%d", haveLen)
		}

		// Compare files byte by byte.
		for i, v := range wantContent {
			if haveContent[i] != v {
				return notice.New("expected file content to match").
					Append("field", "%s", fil.Field).
					Append("filename", "%s", fil.Name).
					Append("first diff index", "%d", i)
			}
		}
	}

	wantLen := len(wantFiles)
	haveLen := len(haveFiles)
	if wantLen != haveLen {
		return notice.New("expected form to have the same number of files").
			Want("%d", wantLen).
			Have("%d", haveLen)
	}

	return nil
}

func (bdy *mpBody) SetContentTypeHeader(h http.Header) {
	bdy.mp.SetContentTypeHeader(h)
}
