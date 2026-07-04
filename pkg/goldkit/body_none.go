package goldkit

import (
	"net/http"

	"github.com/ctx42/testing/pkg/check"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
)

// bodyNone represents golden file's NONE body.
type bodyNone struct{}

// noneBody returns new instance of bodyNone.
func noneBody(body string) (bodyNone, error) {
	if body == "" {
		return bodyNone{}, nil
	}
	return bodyNone{}, notice.New("expected empty body").Have("%s", body)
}

func (bdy bodyNone) Body() []byte { return nil }

func (bdy bodyNone) Assert(t tester.T, have []byte) bool {
	t.Helper()
	if err := check.Empty(have); err != nil {
		err = notice.From(err).
			Prepend("message", "%s", "did not expect body")
		t.Error(err)
		return false
	}
	return true
}

func (bdy bodyNone) SetContentTypeHeader(http.Header) {}
