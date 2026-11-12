package envloader

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/instacart/go-testhooks/testhooks"
)

// StatErrHook allows tests to inject errors from os.Stat without filesystem manipulation.
var StatErrHook = testhooks.NewErrHook()

// Option configures MustLoadEnv.
type Option func(*config)

type config struct {
	requireAll bool
}

// RequireAll returns an Option that makes MustLoadEnv panic when any of the
// given files does not exist. It matches Ruby's Dotenv.load!.
func RequireAll() Option {
	return func(c *config) {
		c.requireAll = true
	}
}

// MustLoadEnv loads the given env files. Relative paths resolve against
// BUILD_WORKING_DIRECTORY when it is set (bazel run), and against the current
// working directory otherwise. That variable is the directory the command was
// invoked from rather than the workspace root, so the paths are relative to
// where the user ran it. Files that do not exist are skipped unless
// RequireAll is passed; any other error panics. Returns true if at least one
// file was loaded.
//
// Loading is skipped once ENV_LOADER_LOADED_ENVS is "true", which the loader
// sets itself so a restarted process does not load the files twice.
func MustLoadEnv(files []string, opts ...Option) bool {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return loadEnv(files, cfg)
}

func loadEnv(files []string, cfg *config) bool {
	if os.Getenv("ENV_LOADER_LOADED_ENVS") == "true" {
		return false
	}

	workingDirectory := os.Getenv("BUILD_WORKING_DIRECTORY")
	var paths []string
	for _, file := range files {
		path := file
		if workingDirectory != "" && !filepath.IsAbs(path) {
			path = filepath.Join(workingDirectory, filepath.FromSlash(path))
		}

		_, err := os.Stat(path)
		err = StatErrHook.Apply(err)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && !cfg.requireAll {
				continue
			}
			panic(err)
		}
		paths = append(paths, path)
	}

	if len(paths) == 0 {
		return false
	}
	if err := godotenv.Overload(paths...); err != nil {
		panic(err)
	}
	_ = os.Setenv("ENV_LOADER_LOADED_ENVS", "true")
	return true
}
