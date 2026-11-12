//go:build windows

package envloader

func restartProcess() {
	panic("envloader: automatic restart is not supported on Windows; pass envloader.NoRestart()")
}
