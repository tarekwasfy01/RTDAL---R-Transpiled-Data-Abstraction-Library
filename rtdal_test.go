package rtdal_test

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"testing"

	rtdal "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library"
)

func ExampleEval() {
	ctx := rtdal.NewContext()
	value, err := rtdal.Eval(ctx, "x <- c(1, 2, 3); sum(x)")
	if err != nil {
		panic(err)
	}
	fmt.Println(value)
	// Output: 6
}

func TestContextAndErrors(t *testing.T) {
	ctx := rtdal.NewContext()
	var output bytes.Buffer
	ctx.Output = &output
	if _, err := rtdal.Eval(ctx, "x <- c(2, 4, 6)"); err != nil {
		t.Fatal(err)
	}
	value, err := rtdal.Eval(ctx, "sum(x)")
	if err != nil || value.String() != "12" {
		t.Fatalf("sum: %v, %v", value, err)
	}
	if _, err := rtdal.Eval(rtdal.NewContext(), "x"); err == nil {
		t.Fatal("contexts share bindings")
	}
	if _, err := rtdal.Eval(ctx, "x <- ("); err == nil {
		t.Fatal("invalid syntax accepted")
	}
	if _, err := rtdal.Eval(ctx, "definitely_missing_function()"); err == nil {
		t.Fatal("missing function accepted")
	}
	if _, err := rtdal.Eval(ctx, "print(42)"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("42")) {
		t.Fatalf("output missing: %q", output.String())
	}
}

func TestTranspile(t *testing.T) {
	code, err := rtdal.Transpile("x <- c(1, 2, 3); print(sum(x))")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", code, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(code, []byte(`"r2go/`)) {
		t.Fatal("generated code still uses local module imports")
	}
	if _, err := rtdal.Transpile("x <- ("); err == nil {
		t.Fatal("invalid source accepted")
	}
}
