package goldkit

import (
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testkit/pkg/oskit"
	"gopkg.in/yaml.v3"
)

func Test_base(t *testing.T) {
	t.Run("minimal", func(t *testing.T) {
		// --- Given ---
		content := oskit.ReadFile(t, "testdata/base_minimal.yml")

		// --- When ---
		gld := &base{}
		err := yaml.Unmarshal(content, gld)

		// --- Then ---
		assert.NoError(t, err)
		wantMeta := map[string]any{
			"key1": "val1",
			"key2": 123,
			"key3": 12.3,
			"key4": time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			"key5": 9223372036854775807,
		}
		assert.Equal(t, wantMeta, gld.Meta)
		assert.Equal(t, "", gld.BodyType)
		assert.Equal(t, "", gld.RawBody.Value)
	})

	t.Run("text body type", func(t *testing.T) {
		// --- Given ---
		content := oskit.ReadFile(t, "testdata/base_text.yml")

		// --- When ---
		gld := &base{}
		err := yaml.Unmarshal(content, gld)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, Text, gld.BodyType)
		assert.Equal(t, "abc", gld.RawBody.Value)
	})

	t.Run("JSON body type", func(t *testing.T) {
		// --- Given ---
		content := oskit.ReadFile(t, "testdata/base_json.yml")

		// --- When ---
		gld := &base{}
		err := yaml.Unmarshal(content, gld)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, JSON, gld.BodyType)
		assert.Equal(t, `{"key2": "val2"}`, gld.RawBody.Value)
	})

	t.Run("custom body type", func(t *testing.T) {
		// --- Given ---
		content := oskit.ReadFile(t, "testdata/base_custom.yml")

		// --- When ---
		gld := &base{}
		err := yaml.Unmarshal(content, gld)

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "custom", gld.BodyType)
		assert.Equal(t, "abc\n", gld.RawBody.Value)
	})
}

func Test_base_M(t *testing.T) {
	// --- Given ---
	want := map[string]any{"key1": "val1"}
	gld := &base{Meta: want}

	// --- When ---
	have := gld.M()

	// --- Then ---
	assert.Same(t, want, have)
}
