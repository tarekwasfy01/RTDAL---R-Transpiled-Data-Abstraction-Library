// Package corpus provides optional access to the translated R source corpus.
// Importing it compiles the large generated corpus; rtdal itself does not.
// Inventory and successful loading do not imply full R-package compatibility.
package corpus

import (
	"fmt"
	"sort"

	rtdal "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/generated"
)

// Package describes available source units, not verified working functions.
type Package struct {
	Name           string
	LibrarySources int
	TotalSources   int
}

// Failure identifies a source unit that could not be loaded.
type Failure = generated.RTDALLoadFailure

// Report includes the context and every skipped source, including partial loads.
type Report struct {
	Context  *rtdal.Context
	Loaded   int
	Failures []Failure
}

// Packages returns a sorted inventory, independent of the working directory.
func Packages() []Package {
	counts := make(map[string]Package)
	for _, unit := range generated.RTDALCorpus {
		p := counts[unit.Package]
		p.Name = unit.Package
		p.TotalSources++
		if unit.Library {
			p.LibrarySources++
		}
		counts[p.Name] = p
	}
	result := make([]Package, 0, len(counts))
	for _, p := range counts {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Load executes only library sources in a fresh context. It returns an error
// for unknown packages, zero successful loads, or any skipped source. On partial
// failure the report still contains the usable context and complete diagnostics.
func Load(name string) (Report, error) {
	found := false
	for _, unit := range generated.RTDALCorpus {
		if unit.Package == name && unit.Library {
			found = true
			break
		}
	}
	if !found {
		return Report{}, fmt.Errorf("no library sources for package %q", name)
	}
	ctx, loaded, failures := generated.LoadRTDALPackageWithReport(name)
	report := Report{Context: ctx, Loaded: loaded, Failures: failures}
	if loaded == 0 || len(failures) > 0 {
		return report, fmt.Errorf("package %q: %d source units loaded, %d skipped", name, loaded, len(failures))
	}
	return report, nil
}
