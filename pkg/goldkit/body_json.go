package goldkit

import (
	"net/http"
	"slices"

	"github.com/ctx42/testing/pkg/check"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
)

// bodyJSON represents golden file's JSON body.
type bodyJSON struct {
	body []byte
}

// jsonBody returns a new instance of [bodyJSON].
func jsonBody(body []byte) *bodyJSON {
	return &bodyJSON{body: body}
}

func (bdy *bodyJSON) Assert(t tester.T, got []byte) bool {
	t.Helper()
	if err := check.JSON(string(bdy.body), string(got)); err != nil {
		t.Error(notice.From(err, "JSON body"))
		return false
	}
	return true
}

func (bdy *bodyJSON) Body() []byte {
	return slices.Clone(bdy.body)
}

func (bdy *bodyJSON) SetContentTypeHeader(h http.Header) {
	h.Set("Content-Type", "application/json")
}
