// Command splitloaders splits large, independent runtime.SetGlobal registration
// sequences into smaller functions to bound Go compiler optimization costs.
// All original statement text, comments, order and context are retained.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	paths, err := filepath.Glob("generated/corpus_part_*.go")
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat("generated/RTDAL_ALL_TRANSPILED.go"); err == nil {
		paths = append(paths, "generated/RTDAL_ALL_TRANSPILED.go")
	}
	for _, path := range paths {
		if err := split(path); err != nil {
			panic(err)
		}
	}
}

func split(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
	if err != nil {
		return err
	}
	offset := func(pos token.Pos) int { return fset.Position(pos).Offset }
	var out strings.Builder
	cursor, changed := 0, false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(fn.Name.Name, "loadRTDAL") || fn.Body == nil || len(fn.Body.List) < 128 {
			continue
		}
		if fn.Recv != nil || fn.Type.Results != nil || len(fn.Type.Params.List) != 1 {
			continue
		}
		param := fn.Type.Params.List[0]
		if len(param.Names) != 1 || param.Names[0].Name != "ctx" {
			continue
		}
		// Only split standalone registrations: no local declarations, control
		// flow or cross-statement Go variable scope can cross a helper boundary.
		safe := true
		for _, statement := range fn.Body.List {
			expr, ok := statement.(*ast.ExprStmt)
			if !ok {
				safe = false
				break
			}
			call, ok := expr.X.(*ast.CallExpr)
			if !ok {
				safe = false
				break
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				safe = false
				break
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "runtime" || sel.Sel.Name != "SetGlobal" {
				safe = false
				break
			}
		}
		if !safe {
			continue
		}
		changed = true
		out.Write(data[cursor : offset(fn.Body.Lbrace)+1])
		const batch = 32
		for i := 0; i < len(fn.Body.List); i += batch {
			fmt.Fprintf(&out, "\n\t%sPart%03d(ctx)", fn.Name.Name, i/batch)
		}
		out.WriteString("\n}\n")
		start := offset(fn.Body.Lbrace) + 1
		for i := 0; i < len(fn.Body.List); i += batch {
			endIndex := min(i+batch, len(fn.Body.List)) - 1
			end := offset(fn.Body.List[endIndex].End())
			if endIndex == len(fn.Body.List)-1 {
				end = offset(fn.Body.Rbrace)
			}
			fmt.Fprintf(&out, "\n//go:noinline\nfunc %sPart%03d(ctx *runtime.Context) {", fn.Name.Name, i/batch)
			out.Write(data[start:end])
			out.WriteString("\n}\n")
			start = end
		}
		cursor = offset(fn.Body.Rbrace) + 1
		fmt.Printf("%s: split %s (%d ordered registrations)\n", path, fn.Name.Name, len(fn.Body.List))
	}
	if !changed {
		return nil
	}
	out.Write(data[cursor:])
	// Validate the complete rewritten Go file before replacing the input.
	if _, err := parser.ParseFile(token.NewFileSet(), path, out.String(), 0); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(out.String()), 0644)
}
