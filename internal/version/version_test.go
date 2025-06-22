package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected Version
		hasError bool
	}{
		{"0.9.0", Version{0, 9, 0}, false},
		{"v0.9.0", Version{0, 9, 0}, false},
		{"1.2.3", Version{1, 2, 3}, false},
		{"v1.2.3", Version{1, 2, 3}, false},
		{"2.0", Version{2, 0, 0}, false},
		{"v2.0", Version{2, 0, 0}, false},
		{"10.20.30", Version{10, 20, 30}, false},
		{"V10.20.30", Version{10, 20, 30}, false},

		// Invalid cases
		{"", Version{}, true},
		{"v", Version{}, true},
		{"1", Version{}, true},
		{"v1", Version{}, true},
		{"0.0.0", Version{0, 0, 0}, true},
		{"v0.0.0", Version{0, 0, 0}, true},
		{"1.2.3.4", Version{}, true},
		{"v1.2.beta", Version{}, true},
		{"abc", Version{}, true},
	}

	for _, tt := range tests {
		v, err := Parse(tt.input)
		if tt.hasError {
			assert.Error(t, err, "expected error for input: %q", tt.input)
		} else {
			assert.NoError(t, err, "unexpected error for input: %q", tt.input)
			assert.Equal(t, tt.expected, v, "parsed version mismatch for input: %q", tt.input)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		v1       Version
		v2       Version
		expected int // -1 = v1 older, 0 = equal, 1 = v1 newer
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{1, 2, 3}, Version{1, 2, 3}, 0},
		{Version{2, 0, 0}, Version{1, 9, 9}, 1},
		{Version{1, 10, 0}, Version{1, 9, 9}, 1},
		{Version{1, 2, 5}, Version{1, 2, 3}, 1},
		{Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{Version{1, 2, 0}, Version{1, 3, 0}, -1},
		{Version{1, 2, 3}, Version{1, 2, 4}, -1},
	}

	for _, tt := range tests {
		result := tt.v1.Compare(tt.v2)
		assert.Equal(t, tt.expected, result, "Compare(%+v, %+v)", tt.v1, tt.v2)
	}
}

func TestVersionIsNewerThan(t *testing.T) {
	assert.True(t, Version{2, 0, 0}.IsNewerThan(Version{1, 9, 9}))
	assert.True(t, Version{1, 2, 3}.IsNewerThan(Version{1, 2, 2}))
	assert.False(t, Version{1, 2, 3}.IsNewerThan(Version{1, 2, 3}))
	assert.False(t, Version{1, 0, 0}.IsNewerThan(Version{1, 1, 0}))
}

func TestVersionIsOlderThan(t *testing.T) {
	assert.True(t, Version{1, 0, 0}.IsOlderThan(Version{1, 1, 0}))
	assert.True(t, Version{1, 2, 2}.IsOlderThan(Version{1, 2, 3}))
	assert.False(t, Version{1, 2, 3}.IsOlderThan(Version{1, 2, 3}))
	assert.False(t, Version{2, 0, 0}.IsOlderThan(Version{1, 9, 9}))
}
