//go:build !unix

package envloader

func restartProcess(string) {
	panic("envloader: automatic restart is not supported on this platform; pass envloader.NoRestart()")
}
