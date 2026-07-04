package goldkit

import (
	"net/http"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/tester"
)

func Test_textBody(t *testing.T) {
	// --- Given ---
	data := "abc"

	// --- When ---
	bdy := textBody(data)

	// --- Then ---
	assert.Equal(t, data, bdy.body)
}

func Test_textBody_Body(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		// --- Given ---
		bdy := textBody("abc")

		// --- When ---
		have := bdy.Body()

		// --- Then ---
		assert.Equal(t, "abc", string(have))
	})

	t.Run("returns copy", func(t *testing.T) {
		// --- Given ---
		bdy := textBody("abc")
		have := bdy.Body()
		have[0] = '['

		// --- When ---
		have = bdy.Body()

		// --- Then ---
		assert.Equal(t, "abc", string(have))
	})
}

func Test_textBody_Assert(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		bdy := textBody("abc")

		// --- When ---
		have := bdy.Assert(tspy, []byte("abc"))

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("fail", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected values to be equal:\n" +
			"  message: TEXT bodies do not match\n" +
			"     want: \"abc\"\n" +
			"     have: \"xyz\""
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy := textBody("abc")

		// --- When ---
		have := bdy.Assert(tspy, []byte("xyz"))

		// --- Then ---
		assert.False(t, have)
	})
}

func Test_textBody_SetContentTypeHeader(t *testing.T) {
	// --- Given ---
	bdy := textBody("abc")
	h := http.Header{
		"Content-Type": []string{"previous value"},
	}

	// --- When ---
	bdy.SetContentTypeHeader(h)

	// --- Then ---
	want := map[string][]string{
		"Content-Type": {"text/plain"},
	}
	assert.Equal(t, want, map[string][]string(h))
}
