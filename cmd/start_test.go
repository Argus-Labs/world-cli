package cmd

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDev(t *testing.T) {
	err := StartCommand(nil, nil)
	assert.NilError(t, err)
}
