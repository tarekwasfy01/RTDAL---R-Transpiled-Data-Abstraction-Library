# RTDAL — Go package

An importable, experimental Pure-Go R compatibility library, transpiler, and
translated package corpus. Requires **Go 1.26 or newer**; no R installation,
cgo, external Go module, or local `replace` directive is required.

## Install from this branch

Run inside your application's Go module:

```sh
go get github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library@GoPackage
```

Go records an immutable pseudo-version for the selected commit in your `go.mod`.
Use `@GoPackage` explicitly: the default branch is not this library release.

## Evaluate R from Go

```go
package main

import (
    "fmt"
    "log"
    rtdal "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library"
)

func main() {
    ctx := rtdal.NewContext()
    value, err := rtdal.Eval(ctx, "x <- c(1, 2, 3); sum(x)")
    if err != nil { log.Fatal(err) }
    fmt.Println(value) // 6
}
```

Reuse the context to retain variables and function definitions. Set `ctx.Output`
to an `io.Writer` to capture evaluator output. Do not share a context across
concurrent calls without synchronization. `NewContextWithHost` accepts a custom
runtime host. **This is not a security sandbox: run trusted R source only.**

## Public API

| API | Purpose |
| --- | --- |
| `rtdal.NewContext()` | Create a persistent R execution context |
| `rtdal.NewContextWithHost(host)` | Customize runtime environment access |
| `rtdal.Eval(ctx, source)` | Execute R source and return `(Value, error)` |
| `rtdal.Transpile(source)` | Generate a Go main package as `([]byte, error)` |
| `rtdal.RunOperation(category, name, args...)` | Invoke GIS operations with CLI-style options |
| `rtdal.LicenseReport()` | Read embedded third-party notices |
| `corpus.Packages()` | List translated packages and source counts |
| `corpus.Load(name)` | Load library sources with explicit failure diagnostics |

Concrete values such as `*runtime.DoubleVector` and the lower-level runtime API
are available at `.../toolchain/r2go/runtime`. Transpiled output uses the same
GitHub module path and builds inside an application that depends on RTDAL.
`RunOperation` retains CLI filesystem/network effects and stdout output.

## Optional translated corpus

Import `github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/corpus`
when you need the full translated corpus:

```go
report, err := corpus.Load("abind")
if err != nil {
    // Includes unknown packages and partially loaded packages.
    // Inspect report.Loaded and report.Failures; do not ignore the error.
    return err
}
value, err := rtdal.Eval(report.Context, "is.function(abind)")
```

The corpus is large and increases download and first-build costs. Importing only
`rtdal` avoids compiling it, but Go still downloads the entire module. The
`generated/internal/part*` packages preserve the local translated corpus in 41
smaller compilation units, all below GitHub's per-file limit. The public registry
remains in `generated/registry.go`. Sources outside package `R/` directories are
inventoried but not executed by `corpus.Load`.

The first corpus build can take several minutes and substantial memory. On
memory-constrained machines, set `GOMEMLIMIT=4GiB` and `GOMAXPROCS=2` before building; this is Go's
soft memory target, not a hard process limit. CI and the Windows build script
set this target. Avoid simultaneous initial corpus builds.

The two largest Torch registration loaders are split into ordered helpers without
changing their registrations; a regression test checks all 6,190 registrations.
After regenerating `generated/RTDAL_ALL_TRANSPILED.go`, run
`go run ./tools/splitloaders` followed by `go run ./tools/shardcorpus` before
building. These tools retain statement order and source text, distribute all
10,477 source-unit loaders, and replace the oversized intermediate files.

## Scope and limitations

This is **not full GNU R or CRAN compatibility**. Translation, primitive routing,
and successful source loading are different from verified behavioral parity.
Unsupported native `.C`, `.Call`, `.External`, graphics, and ABI paths remain
explicit runtime errors. Loading `abind` proves registration, not that every
operation or arbitrary package combination works. The first API is experimental.

## CLI and development

```sh
go test ./...
go vet . ./corpus ./cmd/... ./internal/... ./toolchain/... ./tools/... ./examples/...
go vet -unreachable=false ./...
go build ./...
go run ./examples/eval
go run ./cmd/rtdal help
go run ./cmd/rtdal corpus list
```

The library and CLI build without cgo. The graphical toolbox is Windows-only;
on other systems pass CLI arguments. `build-rtdal-onefile.bat` builds the Windows
executable. Source fetching and corpus regeneration require the original CRAN
sources; ordinary library use and builds do not.

The inherited generated corpus has 13 unreachable-code diagnostics under plain
`go vet ./...` (trailing statements after generated returns). CI runs all vet
checks on the maintained API/runtime/tools and disables only the unreachable-code
analyzer for the full generated corpus. This is an explicit remaining generator
limitation, not an R compatibility guarantee.

## Licences and provenance

The root `LICENSE` covers original RTDAL code (BSD-2-Clause), and
`toolchain/r2go/LICENSE` covers original R2Go code (MIT). These do **not** relicense
translated upstream code. The corpus includes packages declared GPL and other
licences; see `THIRD_PARTY_NOTICES.md`, `LICENSES/`, and the per-package metadata.
GNU R-derived runtime translations also retain their upstream licensing; see
`LICENSES/GNU-R/`. This combined module is not represented as BSD/MIT-only or as
a completed licensing audit. Preserve applicable notices when redistributing.
