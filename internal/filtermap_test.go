package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterMap(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		sut := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		filterMap(sut, "key1")
		assert.Contains(t, sut, "key1")
		assert.NotContains(t, sut, "key2")
	})

	t.Run("complex", func(t *testing.T) {
		sut := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": map[string]interface{}{
				"subkey1": "value1",
				"subkey2": map[string]interface{}{
					"subsubkey1": "value1",
					"subsubkey2": "value2",
				},
			},
			"key4": map[string]interface{}{
				"subkey1": "value1",
			},
		}
		filterMap(sut, "key1", "key3.subkey2.subsubkey2")
		assert.Contains(t, sut, "key1")
		assert.NotContains(t, sut, "key2")
		assert.Contains(t, sut, "key3")
		assert.Contains(t, sut["key3"], "subkey2")
		assert.Contains(t, sut["key3"].(map[string]interface{})["subkey2"], "subsubkey2")
		assert.NotContains(t, sut, "key4")
	})
}
