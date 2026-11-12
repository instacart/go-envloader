# Env Loader (envloader)

Shared utility for loading `.env` files when a service is run locally through a runner that does not start in the project directory (one that sets `BUILD_WORKING_DIRECTORY`, such as `bazel run`).

Pre-1.0, maintained on a best-effort basis. See [Project status](#project-status).

## Problem

Services are commonly run locally through a build/run tool that launches the process from a sandbox rather than from the project directory. Since the process no longer starts in the project directory, the usual `.env` / `.env.personal` files sitting next to the service aren't automatically picked up — they need to be loaded explicitly at startup. To avoid reimplementing this in every service, the common logic is consolidated here:

1. Check whether `BUILD_WORKING_DIRECTORY` is set (i.e. the runner was invoked from a checkout rather than a sandbox)
2. Build absolute paths to the env files relative to that directory
3. Load them with `godotenv.Overload` so they override any existing env vars
4. Set `ENV_LOADER_LOADED_ENVS=true` to prevent double-loading on restart

## Usage

### Typical service

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

Call it at the very top of `main` — before anything else initialises — so that all packages see the loaded environment variables:

```go
func main() {
    myappenvloader.MustLoadEnv()
    // ... rest of startup
}
```

There are no implicit defaults: the caller passes exactly the files it wants, in load order (later files override earlier ones).

### Path resolution and missing files

Relative paths resolve against `BUILD_WORKING_DIRECTORY` when it is set (bazel run), and against the process working directory otherwise. Files that do not exist are skipped, so local setups without `.env.personal` work without extra flags. Pass `RequireAll()` when every listed file must exist. A file that exists but cannot be read (for example a directory) panics either way.

### Use with Bazel

Under `bazel run`, the launcher runs the binary from the sandbox and sets `BUILD_WORKING_DIRECTORY` to the directory the command was typed in. The loader uses that as the base directory, so list files relative to where you run it — which for a service is usually its own directory:

```go
envloader.MustLoadEnv([]string{
    "services/my-service/.env",
    "services/my-service/.env.personal",
})
```

The files are read from the user's checkout rather than from the runfiles tree, so local edits are picked up on the next run.

That base is the invocation directory, **not** the workspace root. Running `bazel run //services/my-service:my-service` from `services/my-service` resolves `.env` to `services/my-service/.env`; running the same command from the workspace root needs the path spelled out. Outside Bazel, the same relative paths resolve against the process working directory instead.

### Process restart after loading

Some services need every subsequently-imported package to see the loaded environment variables. Because Go packages may initialise `init()` functions before `main` runs, the solution is to restart the process via `syscall.Exec` after loading, so the new process inherits the updated environment from the very beginning. This makes calling `MustLoadEnv` as early as possible in `main` especially important.

Restarting is what makes the loaded values visible to packages whose `init()` has already run; without it, those packages keep the environment they saw at first start.

```go
package myappenvloader

import (
    "os"
    "syscall"

    "github.com/instacart/go-envloader/envloader"
    "github.com/instacart/go-testhooks/testhooks"
)

var execFuncHook = testhooks.NewValueHook[func(argv0 string, argv []string, envv []string) error]()

func MustLoadEnv(extraFiles ...string) {
    files := append([]string{
        ".env",
        ".env.personal",
    }, extraFiles...)
    if envloader.MustLoadEnv(files) {
        executable, _ := os.Executable()
        _ = execFuncHook.Apply(syscall.Exec)(executable, os.Args, os.Environ())
    }
}
```

The `execFuncHook` allows tests to intercept the `syscall.Exec` call without actually restarting the process.

## API

```go
// Option configures MustLoadEnv.
type Option func(*config)

// RequireAll makes MustLoadEnv panic when any of the given files does not
// exist. Equivalent of Ruby's Dotenv.load!.
func RequireAll() Option

// MustLoadEnv loads the given env files. Relative paths resolve against
// BUILD_WORKING_DIRECTORY when set, otherwise the working directory.
// Missing files are skipped unless RequireAll is passed; any other error
// panics. Returns true if at least one file was loaded, false if already
// loaded or none of the files exist.
func MustLoadEnv(files []string, opts ...Option) bool
```

`MustLoadEnv`:
- Loads files in the order given, with later files overriding earlier ones
- Skips files that do not exist
- Panics if a file exists but cannot be read
- Is a no-op if `ENV_LOADER_LOADED_ENVS=true` (already ran in this process)

Pass `RequireAll()` to panic on a missing file instead of skipping it.

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
