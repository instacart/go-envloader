//go:build unix

package envloader

import (
	"fmt"
	"os"
	"syscall"

	"github.com/instacart/go-testhooks/testhooks"
)

type execFunc func(argv0 string, argv []string, envv []string) error

// execFuncHook lets tests intercept the process restart. It is intentionally
// unexported so callers cannot replace the process execution behavior.
var execFuncHook = testhooks.NewValueHook[execFunc]()

func restartProcess(executable string) {
	if err := execFuncHook.Apply(syscall.Exec)(executable, os.Args, os.Environ()); err != nil {
		panic(fmt.Errorf("envloader: restart process: %w", err))
	}
}
