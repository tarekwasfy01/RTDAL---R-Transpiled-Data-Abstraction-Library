package generated_test

import (
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/generated"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/runtime"
	"testing"
)

// Check all registrations survive splitting the two large Torch loaders.
// Registration is not a claim that Torch's native algorithms are available.
func TestLargeTorchRegistrationLoaders(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		count  int
	}{
		{"namespace", "sources/torch/R/gen-namespace.R", 2256},
		{"native_wrappers", "sources/torch/R/RcppExports.R", 3934},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := runtime.NewContext()
			found := false
			for _, unit := range generated.RTDALCorpus {
				if unit.Source == tc.source {
					unit.Load(ctx)
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing source: %s", tc.source)
			}
			value, err := runtime.RunSourceInContext(ctx, "length(ls(all.names=TRUE))")
			count, ok := value.(*runtime.IntegerVector)
			if err != nil || !ok || len(count.Data) != 1 || count.Data[0] != int64(tc.count) {
				t.Fatalf("registrations: %v, error=%v; want %d", value, err, tc.count)
			}
		})
	}
}
