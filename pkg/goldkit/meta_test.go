package goldkit

import (
	"testing"
	"time"

	"github.com/ctx42/testing/pkg/assert"
	"github.com/ctx42/testing/pkg/must"
)

var (
	// waw is the Europe/Warsaw timezone.
	waw = must.Value(time.LoadLocation("Europe/Warsaw"))

	// zrh is the Europe/Zurich timezone.
	zrh = must.Value(time.LoadLocation("Europe/Zurich"))
)

func Test_Meta_MetaLookup(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		haveVal, haveExi := Meta(m).MetaLookup("A")

		// --- Then ---
		assert.False(t, haveExi)
		assert.Nil(t, haveVal)
	})

	t.Run("existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"A": 1}

		// --- When ---
		haveVal, haveExi := Meta(m).MetaLookup("A")

		// --- Then ---
		assert.True(t, haveExi)
		assert.Equal(t, 1, haveVal)
	})
}

func Test_Meta_MetaGet(t *testing.T) {
	t.Run("empty collection", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have := Meta(m).MetaGet("A")

		// --- Then ---
		assert.Nil(t, have)
	})

	t.Run("existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"A": 1}

		// --- When ---
		have := Meta(m).MetaGet("A")

		// --- Then ---
		assert.Equal(t, 1, have)
	})
}

func Test_Meta_MetaSet(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have := Meta(m).MetaSet("A", 1)

		// --- Then ---
		assert.Equal(t, map[string]any{"A": 1}, m)
		assert.Same(t, m, have)
	})

	t.Run("set existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"A": 1}

		// --- When ---
		have := Meta(m).MetaSet("A", 2)

		// --- Then ---
		assert.Equal(t, map[string]any{"A": 2}, m)
		assert.Same(t, m, have)
	})
}

func Test_Meta_MetaDelete(t *testing.T) {
	t.Run("delete not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"A": 1, "B": 2, "C": 3}

		// --- When ---
		Meta(m).MetaDelete("D")

		// --- Then ---
		assert.Equal(t, map[string]any{"A": 1, "B": 2, "C": 3}, m)
	})

	t.Run("delete existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"A": 1, "B": 2, "C": 3}

		// --- When ---
		Meta(m).MetaDelete("A")

		// --- Then ---
		assert.Equal(t, map[string]any{"B": 2, "C": 3}, m)
	})
}

func Test_Meta_MetaMeta(t *testing.T) {
	t.Run("existing map", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": map[string]any{"A": 1}}

		// --- When ---
		have, err := Meta(m).MetaMeta("key")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, Meta(map[string]any{"A": 1}), have)
	})

	t.Run("existing MetaMeta", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": Meta{"A": 1}}

		// --- When ---
		have, err := Meta(m).MetaMeta("key")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, Meta(map[string]any{"A": 1}), have)
	})

	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaMeta("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Nil(t, have)
	})

	t.Run("not map", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaMeta("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Nil(t, have)
	})
}

func Test_Meta_MetaGetString(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetString("key")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, "abc", have)
	})

	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetString("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Empty(t, have)
	})

	t.Run("not string type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetString("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Empty(t, have)
	})
}

func Test_Meta_MetaGetBool(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": true}

		// --- When ---
		have, err := Meta(m).MetaGetBool("key")

		// --- Then ---
		assert.NoError(t, err)
		assert.True(t, have)
	})

	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetBool("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.False(t, have)
	})

	t.Run("not bool type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetBool("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.False(t, have)
	})
}

func Test_Meta_MetaGetInt(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetInt("key")

		// --- Then ---
		assert.NoError(t, err)
		assert.Equal(t, 1, have)
	})

	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetInt("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Empty(t, have)
	})

	t.Run("not int type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetInt("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Equal(t, 0, have)
	})
}

func Test_Meta_MetaGetInt64(t *testing.T) {
	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetInt64("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Empty(t, have)
	})

	t.Run("not int64 type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetInt64("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Equal(t, int64(0), have)
	})
}

func Test_Meta_MetaGetInt64_tabular(t *testing.T) {
	tt := []struct {
		testN string

		m    map[string]any
		want int64
	}{
		{"int", map[string]any{"key": 1}, 1},
		{"int8", map[string]any{"key": int8(1)}, 1},
		{"int16", map[string]any{"key": int16(1)}, 1},
		{"int32", map[string]any{"key": int32(1)}, 1},
		{"int64", map[string]any{"key": int64(1)}, 1},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- When ---
			have, err := Meta(tc.m).MetaGetInt64("key")

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.want, have)
		})
	}
}

func Test_Meta_MetaGetFloat64(t *testing.T) {
	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetFloat64("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Empty(t, have)
	})

	t.Run("not float64 type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetFloat64("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Equal(t, float64(0), have)
	})
}

func Test_Meta_MetaGetFloat64_tabular(t *testing.T) {
	tt := []struct {
		testN string

		m    map[string]any
		want float64
	}{
		{"int", map[string]any{"key": 1}, 1},
		{"int8", map[string]any{"key": int8(1)}, 1},
		{"int16", map[string]any{"key": int16(1)}, 1},
		{"int32", map[string]any{"key": int32(1)}, 1},
		{"int64", map[string]any{"key": int64(1)}, 1},
		{"float32", map[string]any{"key": float32(1)}, 1},
		{"float64", map[string]any{"key": float64(1)}, 1},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- When ---
			have, err := Meta(tc.m).MetaGetFloat64("key")

			// --- Then ---
			assert.NoError(t, err)
			assert.Equal(t, tc.want, have)
		})
	}
}

func Test_Meta_MetaGetTime(t *testing.T) {
	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetTime("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})

	t.Run("not time or string type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetTime("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})

	t.Run("parsing error", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetTime("key")

		// --- Then ---
		assert.ErrorIs(t, ErrFormat, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})
}

func Test_Meta_MetaGetTime_tabular(t *testing.T) {
	tim := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	tt := []struct {
		testN string

		m    map[string]any
		want time.Time
	}{
		{"time", map[string]any{"key": tim}, tim},
		{"string time", map[string]any{"key": "2000-01-02T03:04:05Z"}, tim},
		{"zero time", map[string]any{"key": "0000-00-00T00:00:00"}, time.Time{}},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- When ---
			have, err := Meta(tc.m).MetaGetTime("key")

			// --- Then ---
			assert.NoError(t, err)
			assert.Exact(t, tc.want, have)
		})
	}
}

func Test_Meta_MetaGetTimeIn(t *testing.T) {
	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetTimeIn("key", time.UTC)

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})

	t.Run("not time or string type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetTimeIn("key", time.UTC)

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})

	t.Run("parsing error", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetTimeIn("key", time.UTC)

		// --- Then ---
		assert.ErrorIs(t, ErrFormat, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Zero(t, have)
	})
}

func Test_Meta_MetaGetTimeIn_tabular(t *testing.T) {
	tim0 := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	tim1 := time.Date(2000, 1, 2, 3, 4, 5, 0, waw)

	tt := []struct {
		testN string

		m    map[string]any
		tz   *time.Location
		want time.Time
	}{
		{"time UTC", map[string]any{"key": tim0}, time.UTC, tim0},
		{
			"string time UTC",
			map[string]any{"key": "2000-01-02T03:04:05"},
			time.UTC,
			tim0,
		},
		{
			"string time UTC Z",
			map[string]any{"key": "2000-01-02T03:04:05Z"},
			time.UTC,
			tim0,
		},
		{
			"string time WAW",
			map[string]any{"key": "2000-01-02T03:04:05"},
			waw,
			tim1,
		},
		{
			"string time +1 WAW",
			map[string]any{"key": "2000-01-02T03:04:05+01:00"},
			waw,
			tim1,
		},
		{
			"string time +1 WAW to UTC",
			map[string]any{"key": "2000-01-02T03:04:05+01:00"},
			time.UTC,
			tim1.In(time.UTC),
		},
		{
			"zero time",
			map[string]any{"key": "0000-00-00T00:00:00"},
			time.UTC,
			time.Time{}.In(time.UTC),
		},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- When ---
			have, err := Meta(tc.m).MetaGetTimeIn("key", tc.tz)

			// --- Then ---
			assert.NoError(t, err)
			assert.Exact(t, tc.want, have)
		})
	}
}

func Test_Meta_MetaGetLoc(t *testing.T) {
	t.Run("not existing", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{}

		// --- When ---
		have, err := Meta(m).MetaGetLoc("key")

		// --- Then ---
		assert.ErrorIs(t, ErrMissing, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Nil(t, have)
	})

	t.Run("not location or string type", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": 1}

		// --- When ---
		have, err := Meta(m).MetaGetLoc("key")

		// --- Then ---
		assert.ErrorIs(t, ErrType, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Nil(t, have)
	})

	t.Run("parsing error", func(t *testing.T) {
		// --- Given ---
		m := map[string]any{"key": "abc"}

		// --- When ---
		have, err := Meta(m).MetaGetLoc("key")

		// --- Then ---
		assert.ErrorIs(t, ErrFormat, err)
		assert.ErrorContain(t, "`key`", err)
		assert.Nil(t, have)
	})
}

func Test_Meta_MetaGetLoc_tabular(t *testing.T) {
	tt := []struct {
		testN string

		m    map[string]any
		want *time.Location
	}{
		{"string timezone", map[string]any{"key": "Europe/Warsaw"}, waw},
		{"timezone instance", map[string]any{"key": zrh}, zrh},
	}

	for _, tc := range tt {
		t.Run(tc.testN, func(t *testing.T) {
			// --- When ---
			have, err := Meta(tc.m).MetaGetLoc("key")

			// --- Then ---
			assert.NoError(t, err)
			assert.Zone(t, tc.want, have)
		})
	}
}
