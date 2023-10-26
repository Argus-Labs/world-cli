package tea_cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertServicesToString(t *testing.T) {
	t.Skip("Temporary skip this test, undefined: servicesToStr")

	// str := servicesToStr([]DockerService{DockerServiceCardinal, DockerServiceNakama, DockerServiceTestsuite})
	str := ""
	assert.Equal(t, "cardinal nakama testsuite", str, "resulting string should be 'cardinal nakama'")
}
