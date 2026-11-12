# Env Loader (envloader)

Load `.env` files at startup and restart the process so `init()` functions see them.

Pre-1.0, maintained on a best-effort basis. See [Project status](#project-status).

## Usage

Add a thin wrapper in your service's internal `myappenvloader` package that lists the files it wants to load:

```go
package myappenvloader

import "github.com/instacart/go-envloader/envloader"

func MustLoadEnv(extraFiles ...string) bool {
    files := append([]string{
        ".env",
        ".env.personal",
    }, extraFiles...)
    return envloader.MustLoadEnv(files)
}
```

Call it at the very top of `main` — before anything else initializes — so that all packages see the loaded environment variables:

```go
func main() {
    myappenvloader.MustLoadEnv()
    // ... rest of startup
}
```

There are no implicit defaults: the caller passes exactly the files it wants, in load order (later files override earlier ones).

### Path resolution and missing files

Relative paths resolve against `WithBaseDir(dir)`, or against the process working directory when no base directory is set or the base directory is empty. Absolute paths ignore the base directory.

Passing an empty base directory is intentional: `WithBaseDir(os.Getenv("BUILD_WORKING_DIRECTORY"))` uses the variable when a runner provides it and falls back to the working directory when it does not.

Files that do not exist are skipped, so local setups without `.env.personal` work without extra flags. Pass `RequireAll()` when every listed file must exist. A file that exists but cannot be read (for example a directory) panics either way.

### Using with `bazel run`

`bazel run` sets `BUILD_WORKING_DIRECTORY` to the directory the command was run from. Pass it as the base directory:

```go
envloader.MustLoadEnv([]string{
    "services/my-service/.env",
    "services/my-service/.env.personal",
}, envloader.WithBaseDir(os.Getenv("BUILD_WORKING_DIRECTORY")))
```

If `BUILD_WORKING_DIRECTORY` is empty, the loader falls back to the process working directory. `BUILD_WORKSPACE_DIRECTORY` can be used instead when paths should resolve from the repository root.

### Restart after loading

After loading at least one file, `MustLoadEnv` restarts the process with the updated environment by default. This ensures packages whose `init()` functions already ran see the loaded values. The process is re-executed with the same arguments and environment.

Use `NoRestart()` under debuggers (exec detaches Delve or IDE debuggers), in tests, and on Windows. Automatic restart is not supported on Windows; calling `MustLoadEnv` without `NoRestart()` there panics and asks the caller to pass `NoRestart()`.

Loaded files use `godotenv.Overload`, so values in the files override environment variables that were already set.

`ENV_LOADER_LOADED_ENVS` is an internal guard. The loader sets it to `true` before restarting, and the restarted process inherits it so it does not load or restart again. If you set it manually, loading becomes a no-op.

## API

```go
// Option configures MustLoadEnv.
type Option func(*config)

// WithBaseDir sets the directory used to resolve relative paths. An empty
// directory, or omitting the option, uses the process working directory.
func WithBaseDir(dir string) Option

// NoRestart skips the automatic process restart after env files are loaded.
func NoRestart() Option

// RequireAll makes MustLoadEnv panic when any of the given files does not
// exist. Equivalent of Ruby's Dotenv.load!.
func RequireAll() Option

// MustLoadEnv loads the given env files, then restarts the process unless
// NoRestart is passed. Relative paths resolve against WithBaseDir or the
// working directory. Missing files are skipped unless RequireAll is passed;
// any other error panics. Returns true if at least one file was loaded, false
// if already loaded or none of the files exist.
func MustLoadEnv(files []string, opts ...Option) bool
```

`MustLoadEnv`:
- Loads files in the order given, with later files overriding earlier ones
- Skips files that do not exist
- Panics if a file exists but cannot be read
- Is a no-op if `ENV_LOADER_LOADED_ENVS=true` (already ran in this process)

Pass `RequireAll()` to panic on a missing file instead of skipping it. Pass `NoRestart()` to keep the current process running after loading.

## Development

Requires Go 1.25 or later.

```bash
make test
```

The `go-testhooks` dependency is fetched by Go; no manual setup is required.

## Project status

A variant of this library is used at Instacart. We're publishing it so others
can use, adapt, and build on the ideas.

This is not an officially supported Instacart product and comes with no
commitment to ongoing maintenance. Fixes and improvements are made on a
best-effort basis, and the repository may be archived in the future.

The API is pre-1.0 and may change between minor versions; breaking changes
will be noted in the release notes.

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).
