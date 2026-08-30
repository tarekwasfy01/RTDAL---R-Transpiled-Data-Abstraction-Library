// Package rtdal embeds the experimental Pure-Go R compatibility runtime.
// It does not require R or cgo. Unsupported R/native operations return errors;
// it is not a complete implementation of GNU R. Contexts must not be shared
// between goroutines without external synchronization. Evaluate trusted code
// only: the default host has access to the process environment and filesystem.
package rtdal

import (
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/internal/operations"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/compiler"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/runtime"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/syntax"
)

// Context retains variables, functions, output and host state between calls.
type Context = runtime.Context

// Value is an R value. Concrete vector types are available in toolchain/r2go/runtime.
type Value = runtime.Value

// Host controls runtime interactions with the environment. It is not a security sandbox.
type Host = runtime.Host

// NewContext creates an execution context using the local filesystem and stdout.
func NewContext() *Context { return runtime.NewContext() }

// NewContextWithHost creates a context with a custom host. A nil host uses LocalHost.
func NewContextWithHost(host Host) *Context { return runtime.NewContextWithHost(host) }

// Eval executes R source and returns its last value. A nil context creates a fresh
// local context. Reuse a context to preserve variables and function definitions.
func Eval(ctx *Context, source string) (Value, error) {
	return runtime.RunSourceInContext(ctx, source)
}

// Transpile generates a Go main package from R source. Generated code imports
// this module's runtime when native lowering is unavailable. The caller's Go
// module must depend on RTDAL to build that output.
func Transpile(source string) ([]byte, error) {
	program, err := syntax.Parse(source)
	if err != nil {
		return nil, err
	}
	return compiler.GenerateMainWithSource(program, source)
}

// RunOperation invokes a supported GIS operation using CLI-style options,
// for example "--input", "data.json". Operations may write files and stdout
// or use the network, according to their options; errors are returned to Go.
func RunOperation(category, name string, args ...string) error {
	return operations.Run(category, name, args)
}
