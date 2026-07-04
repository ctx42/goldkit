package goldkit

import (
	"net/http"
	"testing"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/tester"
)

func Test_noneBody(t *testing.T) {
	// --- When ---
	_, err := noneBody("abc")

	// --- Then ---
	assert.ErrorContain(t, "expected empty body", err)
}

func Test_noneBody_Body(t *testing.T) {
	// --- Given ---
	bdy := bodyNone{}

	// --- When ---
	have := bdy.Body()

	// --- Then ---
	assert.Nil(t, have)
}

func Test_noneBody_Assert(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.Close()

		bdy := bodyNone{}

		// --- When ---
		have := bdy.Assert(tspy, []byte{})

		// --- Then ---
		assert.True(t, have)
	})

	t.Run("fail", func(t *testing.T) {
		// --- Given ---
		tspy := tester.New(t)
		tspy.ExpectError()
		wMsg := "" +
			"expected argument to be empty:\n" +
			"  message: did not expect body\n" +
			"     want: <empty>\n" +
			"     have: []byte{0x78, 0x79, 0x7a}"
		tspy.ExpectLogEqual(wMsg)
		tspy.Close()

		bdy := bodyNone{}

		// --- When ---
		have := bdy.Assert(tspy, []byte("xyz"))

		// --- Then ---
		assert.False(t, have)
	})
}

func Test_noneBody_SetContentTypeHeader(t *testing.T) {
	// --- Given ---
	bdy := bodyNone{}
	h := http.Header{}

	// --- When ---
	bdy.SetContentTypeHeader(h)

	// --- Then ---
	assert.Empty(t, h)
}
