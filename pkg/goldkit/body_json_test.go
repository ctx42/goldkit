package goldkit

import (
	"net/http"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/tester"
)

func Test_jsonBody(t *testing.T) {
	// --- Given ---
	data := []byte(`{"key":"val"}`)

	// --- When ---
	bdy := jsonBody(data)

	// --- Then ---
	assert.Equal(t, string(data), string(bdy.body))
}

func Test_jsonBody_Assert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		bdy := jsonBody([]byte(`{"key":"val"}`))

		// --- When ---
		have := bdy.Assert(tspy, []byte(`{"key":"val"}`))

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("fail", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"[JSON body] expected JSON strings to be equal:\n" +
			"  want: {\"key\":\"val\"}\n" +
			"  have: {\"key\":\"val2\"}"
		tspy.ExpectLogContain(wMsg)
		tspy.Close()

		bdy := jsonBody([]byte(`{"key":"val"}`))

		// --- When ---
		have := bdy.Assert(tspy, []byte(`{"key":"val2"}`))

		// --- Then ---
		assert.False(t, have)
	})
}

func Test_jsonBody_Body(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		// --- Given ---
		bdy := jsonBody([]byte(`{"key":"val"}`))

		// --- When ---
		got := bdy.Body()

		// --- Then ---
		assert.Equal(t, `{"key":"val"}`, string(got))
	})

	t.Run("returns copy", func(t *testing.T) {
		// --- Given ---
		bdy := jsonBody([]byte(`{"key":"val"}`))
		got := bdy.Body()
		got[0] = '['

		// --- When ---
		got = bdy.Body()

		// --- Then ---
		assert.Equal(t, `{"key":"val"}`, string(got))
	})
}

func Test_jsonBody_SetContentTypeHeader(t *testing.T) {
	// --- Given ---
	bdy := jsonBody([]byte(`{"key":"val"}`))
	h := http.Header{
		"Content-Type": []string{"previous value"},
	}

	// --- When ---
	bdy.SetContentTypeHeader(h)

	// --- Then ---
	exp := []string{"application/json"}
	assert.Equal(t, exp, h.Values("Content-Type"))
}
