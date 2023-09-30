package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
)

func RunShellCommandReturnBuffers(cmd string, cap uint) (stdoutBuffer *LogByteBuffer, stderrBuffer *LogByteBuffer, proc *exec.Cmd, err error) {
	stdoutBuffer = NewLogByteBuffer(cap)
	stderrBuffer = NewLogByteBuffer(cap)
	proc, err = PipeShellCmdPipeToBuffers(cmd, stdoutBuffer, stderrBuffer)
	if err != nil {
		stdoutBuffer = nil
		stderrBuffer = nil
		proc = nil
	}
	return
}

func PipeShellCmdPipeToBuffers(cmd string, stdoutBuffer io.Writer, stderrBuffer io.Writer) (*exec.Cmd, error) {
	result := exec.Command(cmd)
	if result.Err != nil {
		return nil, result.Err
	}
	result.Stdout = stdoutBuffer
	result.Stderr = stderrBuffer
	go func() {
		_ = result.Run()
	}()
	return result, nil
}

// RunShellCmd run a command using the shell; no need to split args
// from https://stackoverflow.com/questions/6182369/exec-a-shell-command-in-go
func RunShellCmd(cmd string, shell bool, stdout bool) {
	logger := log.New(os.Stderr, "", 0)
	var pending *exec.Cmd
	if shell {
		pending = exec.Command("bash", "-c", cmd)
	} else {
		pending = exec.Command(cmd)
	}
	if stdout {
		pending.Stderr = os.Stderr
		pending.Stdout = os.Stdout
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		s := <-signalChan
		if pending.Process != nil {
			if err := pending.Process.Signal(s); err != nil {
				panic(fmt.Sprintf("Error forwarding signal: %v\n", err))
			}
		}
	}()

	err := pending.Run()
	if err != nil {
		switch v := err.(type) {
		case *exec.ExitError:
			logger.Fatalf("%s", string(v.Stderr))
		default:
			logger.Fatalf(err.Error())
		}
	}

}
