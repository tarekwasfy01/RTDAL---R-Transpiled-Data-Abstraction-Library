package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/internal/transpile"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	results, err := transpile.Run(root, 8)
	if err != nil {
		fail(err)
	}
	manifest := filepath.Join(root, "manifests", "transpile_matrix.csv")
	if err := transpile.WriteManifest(manifest, results); err != nil {
		fail(err)
	}
	removed := filepath.Join(root, "manifests", "removed_functions.csv")
	if err := transpile.WriteRemovedManifest(removed, root, results); err != nil {
		fail(err)
	}
	aggregate := filepath.Join(root, "generated", "RTDAL_ALL_TRANSPILED.go")
	if err := transpile.WriteAggregate(root, aggregate, results); err != nil {
		fail(err)
	}
	counts := map[string]int{}
	blocks, matrix, compatibility := 0, 0, 0
	for _, result := range results {
		counts[result.Status]++
		blocks += result.TopLevelBlocks
		matrix += result.MatrixBlocks
		compatibility += result.CompatibilityBlocks
	}
	fmt.Printf("TRANSPILE MATRIX files=%d generated=%d compatibility_files=%d parse_failed=%d generate_failed=%d panic=%d\n", len(results), counts["generated"], counts["generated-with-compatibility"], counts["parse-failed"], counts["generate-failed"], counts["panic"])
	fmt.Printf("BLOCK MATRIX total=%d native=%d compatibility=%d output=%s\n", blocks, matrix, compatibility, manifest)
	fmt.Printf("AGGREGATE PASS output=%s\n", aggregate)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "RTDAL TRANSPILE FAIL:", err)
	os.Exit(1)
}
