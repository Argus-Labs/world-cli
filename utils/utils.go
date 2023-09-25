package utils

import (
	"log"
	"os"
	"os/exec"
)

// RunShellCmd run a command using the shell; no need to split args
// from https://stackoverflow.com/questions/6182369/exec-a-shell-command-in-go
func RunShellCmd(cmd string, shell bool) []byte {
	logger := log.New(os.Stderr, "", 0)
	if shell {
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			switch v := err.(type) {
			case *exec.ExitError:
				logger.Fatalf("%s", string(v.Stderr))
			default:
				logger.Fatalf(err.Error())
			}
		}
		return out
	}
	out, err := exec.Command(cmd).Output()
	if err != nil {
		log.Fatal(err)
	}
	return out
}
