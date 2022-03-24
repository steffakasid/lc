package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	t.Run("Return true", func(t *testing.T) {
		res, _ := contains([]string{"a", "b"}, "a")
		assert.True(t, res)
	})

	t.Run("Return false", func(t *testing.T) {
		res, _ := contains([]string{"a", "b"}, "c")
		assert.False(t, res)
	})

	t.Run("Complex", func(t *testing.T) {
		res, subfilter := contains([]string{"a.b", "c"}, "a")
		assert.True(t, res)
		assert.Equal(t, "b", subfilter)
	})
}
