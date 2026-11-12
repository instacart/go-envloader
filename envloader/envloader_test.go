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

	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func load(t *testing.T, files []string, opts ...envloader.Option) bool {
	t.Helper()
	return envloader.MustLoadEnv(files, append(opts, envloader.NoRestart())...)
}

func TestMustLoadEnv(t *testing.T) {
	t.Run("LoadsExistingFiles", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{
			"a.env": "TEST_A=1\n",
			"b.env": "TEST_B=2\n",
		})
		t.Setenv("TEST_A", "")
		t.Setenv("TEST_B", "")

		assert.True(t, load(t, []string{"a.env", "b.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
		assert.Equal(t, "2", os.Getenv("TEST_B"))
		assert.Equal(t, "true", os.Getenv("ENV_LOADER_LOADED_ENVS"))
	})

	t.Run("LaterFilesOverrideEarlierOnes", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{
			"a.env": "TEST_A=first\n",
			"b.env": "TEST_A=second\n",
		})
		t.Setenv("TEST_A", "")

		assert.True(t, load(t, []string{"a.env", "b.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "second", os.Getenv("TEST_A"))
	})

	t.Run("SkipsMissingFiles", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")

		assert.True(t, load(t, []string{"missing.env", "a.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("NoExistingFilesLoadsNothing", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)

		assert.False(t, load(t, []string{"missing.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "", os.Getenv("ENV_LOADER_LOADED_ENVS"))
	})

	t.Run("SkipsOnceLoaded", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")
		t.Setenv("ENV_LOADER_LOADED_ENVS", "true")

		assert.False(t, load(t, []string{"a.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "", os.Getenv("TEST_A"))
	})

	t.Run("BaseDirUnsetUsesWorkingDirectory", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")
		t.Chdir(dir)

		assert.True(t, load(t, []string{"a.env"}))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("BaseDirEmptyUsesWorkingDirectory", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")
		t.Chdir(dir)

		assert.True(t, load(t, []string{"a.env"}, envloader.WithBaseDir("")))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("BaseDirExplicit", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")

		assert.True(t, load(t, []string{"a.env"}, envloader.WithBaseDir(dir)))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("AbsolutePathsIgnoreBaseDir", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")

		assert.True(t, load(t, []string{filepath.Join(dir, "a.env")}, envloader.WithBaseDir(t.TempDir())))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("PanicsOnUnreadableFile", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)
		require.NoError(t, os.Mkdir(filepath.Join(dir, "a.env"), 0o755))

		assert.Panics(t, func() {
			load(t, []string{"a.env"}, envloader.WithBaseDir(dir))
		})
	})

	t.Run("PanicsOnStatErrorOtherThanNotExist", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		envloader.StatErrHook.WithError(errors.New("simulated permission denied")).SetupInTest(t)

		assert.Panics(t, func() {
			load(t, []string{"a.env"}, envloader.WithBaseDir(dir))
		})
	})

	t.Run("RequireAllLoadsWhenAllFilesExist", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("TEST_A", "")

		assert.True(t, load(t, []string{"a.env"}, envloader.WithBaseDir(dir), envloader.RequireAll()))
		assert.Equal(t, "1", os.Getenv("TEST_A"))
	})

	t.Run("RequireAllPanicsOnMissingFile", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})

		assert.Panics(t, func() {
			load(t, []string{"a.env", "missing.env"}, envloader.WithBaseDir(dir), envloader.RequireAll())
		})
	})

	t.Run("RequireAllSkipsOnceLoaded", func(t *testing.T) {
		dir := setUpWorkingDir(t, map[string]string{"a.env": "TEST_A=1\n"})
		t.Setenv("ENV_LOADER_LOADED_ENVS", "true")

		assert.False(t, load(t, []string{"a.env"}, envloader.WithBaseDir(dir), envloader.RequireAll()))
	})

	t.Run("NoFilesLoadsNothing", func(t *testing.T) {
		dir := setUpWorkingDir(t, nil)

		assert.False(t, load(t, nil, envloader.WithBaseDir(dir)))
	})
}
