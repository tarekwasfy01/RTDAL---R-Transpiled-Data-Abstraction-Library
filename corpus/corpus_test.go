package corpus_test

import (
	rtdal "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/corpus"
	"testing"
)

func TestInventoryAndLoad(t *testing.T) {
	packages := corpus.Packages()
	if len(packages) == 0 {
		t.Fatal("empty inventory")
	}
	total := 0
	for i, p := range packages {
		total += p.TotalSources
		if p.TotalSources < p.LibrarySources {
			t.Fatalf("bad counts: %+v", p)
		}
		if i > 0 && packages[i-1].Name >= p.Name {
			t.Fatal("unsorted inventory")
		}
	}
	if total != 10477 {
		t.Fatalf("source units=%d, want 10477", total)
	}
	if _, err := corpus.Load("does-not-exist"); err == nil {
		t.Fatal("unknown package accepted")
	}
	report, err := corpus.Load("abind")
	if err != nil {
		t.Fatalf("abind: %v; %+v", err, report.Failures)
	}
	value, err := rtdal.Eval(report.Context, "is.function(abind)")
	if err != nil || value.String() != "TRUE" {
		t.Fatalf("abind binding: %v, %v", value, err)
	}
}
