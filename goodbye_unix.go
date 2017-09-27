// +build !windows

package goodbye

import (
	"os"
	"syscall"
)

func init() {
	defaultSignals = map[os.Signal]int{
		syscall.SIGKILL: 1,
		syscall.SIGHUP:  0,
		syscall.SIGINT:  0,
		syscall.SIGQUIT: 0,
		syscall.SIGTERM: 0,
	}
}
