package tea_cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertServicesToString(t *testing.T) {
	str := servicesToStr([]DockerService{DockerServiceCardinal, DockerServiceNakama, DockerServiceTestsuite})
	assert.Equal(t, "cardinal nakama testsuite", str, "resulting string should be 'cardinal nakama'")
}
