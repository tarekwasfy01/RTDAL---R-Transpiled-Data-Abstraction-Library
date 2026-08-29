package command

import (
	"fmt"
	"sort"
	"strings"
)

type State string

const (
	Available State = "available"
	Planned   State = "planned"
)

type Spec struct {
	Category string
	Name     string
	Usage    string
	Summary  string
	State    State
}

func Catalog() []Spec {
	result := []Spec{
		{"corpus", "list", "", "List every embedded transpiled package and its source-unit count.", Available},
		{"corpus", "sources", "--package <name>", "List embedded transpiled source units for a package.", Available},
		{"corpus", "load", "--package <name>", "Load all transpiled source units for a package into the Pure-Go runtime.", Available},
		{"source", "fetch", "[package ...]", "Download exact package sources from the local DESCRIPTION inventory.", Available},
		{"source", "fetch-all", "", "Download every available package source and write the fetch matrix.", Available},
		{"source", "inventory", "", "Build package, native-boundary, dependency, and licence matrices.", Available},
		{"source", "licenses", "", "Regenerate per-package licence and provenance files.", Available},
		{"transpile", "corpus", "", "Transpile every R file and build the combined Go corpus.", Available},
	}
	planned := map[string][][3]string{
		"vector": {
			{"info", "--input <dataset> [--layer <name>] [--json]", "Inspect layers, fields, geometry, extent, and CRS."},
			{"convert", "--input <dataset> --output <dataset> [--format <driver>]", "Convert vector datasets."},
			{"bbox", "--input <dataset> [--layer <name>]", "Calculate a vector bounding box."},
			{"reproject", "--input <dataset> --output <dataset> --target-crs <crs>", "Reproject vector geometries."},
			{"buffer", "--input <dataset> --output <dataset> --distance <value>", "Create geometry buffers."},
			{"simplify", "--input <dataset> --output <dataset> --tolerance <value>", "Simplify geometries."},
			{"centroid", "--input <dataset> --output <dataset>", "Calculate geometry centroids."},
			{"area", "--input <dataset> [--geodesic]", "Calculate feature areas."},
			{"length", "--input <dataset> [--geodesic]", "Calculate feature lengths."},
			{"distance", "--input <dataset> --other <dataset>", "Calculate geometry distances."},
			{"intersect", "--input <dataset> --other <dataset> --output <dataset>", "Intersect vector layers."},
			{"union", "--input <dataset> [--other <dataset>] --output <dataset>", "Union vector geometries."},
			{"difference", "--input <dataset> --other <dataset> --output <dataset>", "Subtract geometries."},
			{"clip", "--input <dataset> --mask <dataset> --output <dataset>", "Clip a vector dataset."},
			{"validate", "--input <dataset> [--repair] [--output <dataset>]", "Validate or repair geometries."},
		},
		"raster": {
			{"info", "--input <dataset> [--json]", "Inspect size, bands, type, extent, transform, and CRS."},
			{"convert", "--input <dataset> --output <dataset> [--format <driver>]", "Convert raster datasets."},
			{"crop", "--input <dataset> --output <dataset> --bbox <xmin,ymin,xmax,ymax>", "Crop a raster."},
			{"resample", "--input <dataset> --output <dataset> --resolution <x,y>", "Resample a raster."},
			{"reproject", "--input <dataset> --output <dataset> --target-crs <crs>", "Warp a raster to another CRS."},
			{"mosaic", "--input <dataset...> --output <dataset>", "Mosaic raster datasets."},
			{"calc", "--input <dataset...> --output <dataset> --expression <expr>", "Evaluate a raster expression."},
			{"statistics", "--input <dataset> [--band <n>] [--json]", "Calculate raster statistics."},
			{"tile", "--input <dataset> --output <directory> --zoom <min:max>", "Generate raster tiles."},
		},
		"cube": {
			{"info", "--input <dataset> [--json]", "Inspect spatiotemporal dimensions and attributes."},
			{"subset", "--input <dataset> --output <dataset> [--select <slice>]", "Subset a spatiotemporal data cube."},
			{"aggregate", "--input <dataset> --output <dataset> --by <dimension> --function <name>", "Aggregate a data cube."},
			{"merge", "--input <dataset...> --output <dataset> --along <dimension>", "Merge compatible data cubes."},
			{"warp", "--input <dataset> --output <dataset> --target <grid>", "Warp a data cube to a target grid."},
		},
		"stac": {
			{"collections", "--url <endpoint> [--json]", "List STAC collections."},
			{"search", "--url <endpoint> [--bbox <bbox>] [--datetime <range>] [--query <expr>]", "Search STAC items."},
			{"assets", "--input <item.json> [--json]", "List assets from STAC items."},
			{"download", "--input <item.json> --output <directory> [--asset <key>]", "Download selected STAC assets."},
		},
		"gtfs": {
			{"info", "--input <feed.zip> [--json]", "Inspect a GTFS feed."},
			{"validate", "--input <feed.zip> [--json]", "Validate GTFS tables and references."},
			{"subset", "--input <feed.zip> --output <feed.zip> [--route <id>]", "Subset a GTFS feed."},
			{"export", "--input <feed.zip> --output <directory>", "Export GTFS tables."},
		},
		"geodesy": {
			{"distance", "--from <lon,lat> --to <lon,lat> [--method <name>]", "Calculate geodesic distance."},
			{"bearing", "--from <lon,lat> --to <lon,lat>", "Calculate initial bearing."},
			{"destination", "--from <lon,lat> --bearing <degrees> --distance <metres>", "Calculate a destination point."},
			{"polygon-area", "--input <dataset> [--layer <name>]", "Calculate geodesic polygon area."},
		},
		"classify": {
			{"equal", "--input <values> --classes <n>", "Create equal-width class intervals."},
			{"quantile", "--input <values> --classes <n>", "Create quantile class intervals."},
			{"jenks", "--input <values> --classes <n>", "Create Jenks natural breaks."},
			{"pretty", "--input <values> --classes <n>", "Create rounded class intervals."},
		},
	}
	for category, commands := range planned {
		for _, item := range commands {
			result = append(result, Spec{Category: category, Name: item[0], Usage: item[1], Summary: item[2], State: Planned})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Name < result[j].Name
	})
	return result
}

func Find(category, name string) (Spec, bool) {
	for _, spec := range Catalog() {
		if spec.Category == category && spec.Name == name {
			return spec, true
		}
	}
	return Spec{}, false
}

func Help(filters ...string) string {
	var output strings.Builder
	output.WriteString("RTDAL - R-Transpiled Data Abstraction Library\n\n")
	output.WriteString("Usage: rtdal <category> <command> [options]\n")
	output.WriteString("       rtdal help [category [command]]\n")
	output.WriteString("       rtdal --license\n\n")
	if len(filters) >= 2 {
		if spec, ok := Find(filters[0], filters[1]); ok {
			fmt.Fprintf(&output, "%s %s [%s]\n\nUsage: rtdal %s %s %s\n\n%s\n", spec.Category, spec.Name, spec.State, spec.Category, spec.Name, spec.Usage, spec.Summary)
			return output.String()
		}
	}
	current := ""
	for _, spec := range Catalog() {
		if len(filters) == 1 && spec.Category != filters[0] {
			continue
		}
		if spec.Category != current {
			current = spec.Category
			fmt.Fprintf(&output, "%s:\n", strings.ToUpper(current))
		}
		fmt.Fprintf(&output, "  %-18s %-11s %s\n", spec.Name, "["+string(spec.State)+"]", spec.Summary)
	}
	output.WriteString("\nGlobal options: --help, --version, --license\n")
	return output.String()
}
