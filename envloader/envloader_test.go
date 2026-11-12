package envloader_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/instacart/go-envloader/envloader"
)

func setUpWorkingDir(t *testing.T, files map[string]string) string {
	t.Helper()

	t.Setenv("ENV_LOADER_LOADED_ENVS", "")
	t.Setenv("BUILD_WORKING_DIRECTORY", "")

	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func TestMustLoadEnv(t *testing.T) {
	t.Run("LoadsExistingFiles", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{
			"a.env": "TEST_A=1\n",
			"b.env": "TEST_B=2\n",
		})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("TEST_A", "")
		t.Setenv("TEST_B", "")

		assert.True(t, envloader.MustLoadEnv([]string{"a.env", "b.env"}))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
		assert.Equal(t, "2", os.Getenv("TEST_B"))
		assert.Equal(t, "true", os.Getenv("ENV_LOADER_LOADED_ENVS"))
	})

	t.Run("LaterFilesOverrideEarlierOnes", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{
			"a.env": "TEST_A=first\n",
			"b.env": "TEST_A=second\n",
		})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("TEST_A", "")

		assert.True(t, envloader.MustLoadEnv([]string{"a.env", "b.env"}))
		assert.Equal(t, "second", os.Getenv("TEST_A"))
	})

	t.Run("SkipsMissingFiles", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("TEST_A", "")

		assert.True(t, envloader.MustLoadEnv([]string{"missing.env", "a.env"}))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("NoExistingFilesLoadsNothing", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)

		assert.False(t, envloader.MustLoadEnv([]string{"missing.env"}))
		assert.Equal(t, "", os.Getenv("ENV_LOADER_LOADED_ENVS"))
	})

	t.Run("SkipsOnceLoaded", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("TEST_A", "")
		t.Setenv("ENV_LOADER_LOADED_ENVS", "true")

		assert.False(t, envloader.MustLoadEnv([]string{"a.env"}))
		assert.Equal(t, "", os.Getenv("TEST_A"))
	})

	t.Run("ResolvesAgainstProcessWorkingDirectoryWhenUnset", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")
		t.Chdir(dir)

		assert.True(t, envloader.MustLoadEnv([]string{"a.env"}))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("AbsolutePathsWork", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")

		assert.True(t, envloader.MustLoadEnv([]string{filepath.Join(dir, "a.env")}))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("PanicsOnUnreadableFile", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		require.NoError(t, os.Mkdir(filepath.Join(dir, "a.env"), 0o755))

		assert.Panics(t, func() { envloader.MustLoadEnv([]string{"a.env"}) })
	})

	t.Run("PanicsOnStatErrorOtherThanNotExist", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		envloader.StatErrHook.WithError(errors.New("simulated permission denied")).SetupInTest(t)

		assert.Panics(t, func() { envloader.MustLoadEnv([]string{"a.env"}) })
	})

	t.Run("RequireAllLoadsWhenAllFilesExist", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("TEST_A", "")

		assert.True(t, envloader.MustLoadEnv([]string{"a.env"}, envloader.RequireAll()))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("RequireAllPanicsOnMissingFile", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)

		assert.Panics(t, func() {
			envloader.MustLoadEnv([]string{"a.env", "missing.env"}, envloader.RequireAll())
		})
	})

	t.Run("RequireAllSkipsOnceLoaded", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)
		t.Setenv("ENV_LOADER_LOADED_ENVS", "true")

		assert.False(t, envloader.MustLoadEnv([]string{"a.env"}, envloader.RequireAll()))
	})

	t.Run("NoFilesLoadsNothing", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)
		t.Setenv("BUILD_WORKING_DIRECTORY", dir)

		assert.False(t, envloader.MustLoadEnv(nil))
	})

	// The two variables point at different directories under bazel run:
	// BUILD_WORKING_DIRECTORY is where the command was typed, and
	// BUILD_WORKSPACE_DIRECTORY is the workspace root. Only the first is the
	// base for these paths, so setting the other must not find the file.
	t.Run("IgnoresBuildWorkspaceDirectory", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", dir)
		t.Chdir(t.TempDir())

		assert.False(t, envloader.MustLoadEnv([]string{"a.env"}))
		assert.Equal(t, "", os.Getenv("TEST_A"))
	})
}
