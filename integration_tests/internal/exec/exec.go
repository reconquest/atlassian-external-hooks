package exec

import (
	"fmt"
	"os/exec"
	"sync/atomic"

	"github.com/reconquest/lexec-go"
	"github.com/reconquest/pkg/log"
)

var (
	counter int32
)

func New(command string, args ...string) *lexec.Execution {
	id := atomic.AddInt32(&counter, 1)

	return lexec.NewExec(
		lexec.Loggerf(
			func(message string, args ...interface{}) {
				log.NewChildWithPrefix(
					fmt.Sprintf("{exec} (#%04d) %s:", id, command),
				).Tracef(nil, message, args...)
			},
		),
		exec.Command(command, args...),
	)
}
