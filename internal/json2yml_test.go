package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJson2Yaml(t *testing.T) {
	str, err := json2yaml("{\"test\": \"test\"}")
	assert.NoError(t, err)
	assert.Equal(t, "test: test\n", str)
}
