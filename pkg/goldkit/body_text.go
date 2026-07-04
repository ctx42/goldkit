package goldkit

import (
	"net/http"

	"github.com/ctx42/testing/pkg/check"
	"github.com/ctx42/testing/pkg/notice"
	"github.com/ctx42/testing/pkg/tester"
)

// bodyText represents golden file's TEXT body.
type bodyText struct {
	body string
}

// textBody returns new instance of bodyText.
func textBody(body string) *bodyText {
	return &bodyText{body: body}
}

func (bdy *bodyText) Body() []byte {
	return []byte(bdy.body)
}

func (bdy *bodyText) Assert(t tester.T, have []byte) bool {
	t.Helper()
	if err := check.Equal(bdy.body, string(have)); err != nil {
		err = notice.From(err).
			Prepend("message", "%s", "TEXT bodies do not match")
		t.Error(err)
		return false
	}
	return true
}

func (bdy *bodyText) SetContentTypeHeader(h http.Header) {
	h.Set("Content-Type", "text/plain")
}
