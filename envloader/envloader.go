package envloader

import (
	"errors"
	"fmt"
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
	noRestart  bool
	baseDir    string
}

// RequireAll returns an Option that makes MustLoadEnv panic when any of the
// given files does not exist. It matches Ruby's Dotenv.load!.
func RequireAll() Option {
	return func(c *config) {
		c.requireAll = true
	}
}

// WithBaseDir sets the directory that relative file paths are resolved against.
// An empty directory, or no WithBaseDir option, uses the current working
// directory. Absolute paths ignore the base directory.
//
// This makes it possible to write
// WithBaseDir(os.Getenv("BUILD_WORKING_DIRECTORY")): an empty environment
// variable falls back to the process working directory, while Bazel supplies
// the directory the command was run from.
func WithBaseDir(dir string) Option {
	return func(c *config) {
		c.baseDir = dir
	}
}

// NoRestart disables the process restart after env files are loaded. Use it
// when running under a debugger, in tests, or on Windows.
func NoRestart() Option {
	return func(c *config) {
		c.noRestart = true
	}
}

// MustLoadEnv loads the given env files. Relative paths resolve against
// WithBaseDir, or the current working directory when no base directory is set.
// Files that do not exist are skipped unless RequireAll is passed; any other
// error panics. Returns true if at least one file was loaded.
//
// After loading at least one file, MustLoadEnv restarts the process so packages
// whose init functions already ran see the new environment. Pass NoRestart to
// skip that restart.
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

	baseDir := cfg.baseDir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			panic(fmt.Errorf("envloader: resolve working directory: %w", err))
		}
	}

	var paths []string
	for _, file := range files {
		path := file
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, filepath.FromSlash(path))
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

	if !cfg.noRestart {
		restartProcess()
	}
	return true
}
