//go:build !unix

package envloader

func restartProcess() {
	panic("envloader: automatic restart is not supported on this platform; pass envloader.NoRestart()")
}
