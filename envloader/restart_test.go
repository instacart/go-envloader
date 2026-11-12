//go:build !windows

package envloader

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRestartEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("TEST_RESTART_VALUE=1\n"), 0o644))
	return dir
}

func TestMustLoadEnvRestartsByDefault(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	dir := writeRestartEnv(t)

	var gotArgv0 string
	var gotArgv, gotEnv []string
	execFuncHook.WithValue(func(argv0 string, argv []string, envv []string) error {
		gotArgv0 = argv0
		gotArgv = argv
		gotEnv = envv
		return nil
	}).SetupInTest(t)

	assert.True(t, MustLoadEnv([]string{".env"}, WithBaseDir(dir)))
	assert.NotEmpty(t, gotArgv0)
	assert.Equal(t, os.Args, gotArgv)
	assert.Contains(t, gotEnv, "TEST_RESTART_VALUE=1")
	assert.Contains(t, gotEnv, "ENV_LOADER_LOADED_ENVS=true")
}

func TestNoRestartSkipsExec(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	dir := writeRestartEnv(t)

	execFuncHook.WithValue(func(string, []string, []string) error {
		t.Fatal("exec must not run with NoRestart")
		return nil
	}).SetupInTest(t)

	assert.True(t, MustLoadEnv([]string{".env"}, WithBaseDir(dir), NoRestart()))
}

func TestNoExecWhenNoFilesLoaded(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	dir := t.TempDir()

	execFuncHook.WithValue(func(string, []string, []string) error {
		t.Fatal("exec must not run when no file was loaded")
		return nil
	}).SetupInTest(t)

	assert.False(t, MustLoadEnv([]string{"missing.env"}, WithBaseDir(dir)))
}

func TestNoExecWhenAlreadyLoaded(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "true")
	dir := writeRestartEnv(t)

	execFuncHook.WithValue(func(string, []string, []string) error {
		t.Fatal("exec must not run when loading is already complete")
		return nil
	}).SetupInTest(t)

	assert.False(t, MustLoadEnv([]string{".env"}, WithBaseDir(dir)))
}

func TestExecFailurePanics(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	dir := writeRestartEnv(t)

	execFuncHook.WithValue(func(string, []string, []string) error {
		return errors.New("exec failed")
	}).SetupInTest(t)

	assert.Panics(t, func() {
		MustLoadEnv([]string{".env"}, WithBaseDir(dir))
	})
}

func TestRestartPassesLoadedEnvironment(t *testing.T) {
	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	dir := writeRestartEnv(t)

	var gotEnv []string
	execFuncHook.WithValue(func(_ string, _ []string, envv []string) error {
		gotEnv = envv
		return nil
	}).SetupInTest(t)

	MustLoadEnv([]string{".env"}, WithBaseDir(dir))
	assert.True(t, slices.Contains(gotEnv, "TEST_RESTART_VALUE=1"))
}
