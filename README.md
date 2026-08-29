# RTDAL

RTDAL is a Pure-Go, R-transpiled data abstraction toolbox for GIS. The uniform command
syntax is:

```text
rtdal <category> <command> [options]
```

`available` commands work in the current build. `planned` commands have a
stable, documented CLI contract but are not presented as implemented GIS work.

## Global commands

| Command | What it does |
| --- | --- |
| `rtdal help` | Lists every command grouped by category. |
| `rtdal help <category>` | Lists commands in one category. |
| `rtdal help <category> <command>` | Shows syntax and a concise explanation for one command. |
| `rtdal --version` | Prints the RTDAL version. |
| `rtdal --license` | Lists bundled upstream licences and source provenance. |
| `rtdal --gui` | Opens the RTDAL desktop toolbox. |

## Corpus and source pipeline

| Command | What it does |
| --- | --- |
| `rtdal corpus list` | Lists embedded transpiled packages and their library-source counts. |
| `rtdal corpus sources --package <name>` | Lists every embedded source unit for a package. |
| `rtdal corpus load --package <name>` | Loads library units into the Pure-Go runtime; isolated failures are reported and skipped. |
| `rtdal source fetch [package ...]` | Downloads the exact CRAN source archive(s) from the local package inventory. |
| `rtdal source fetch-all` | Downloads all available package archives and writes the fetch matrix. |
| `rtdal source inventory` | Builds the source, native-boundary, dependency and licence inventory. |
| `rtdal source licenses` | Regenerates preserved package licence and provenance material. |
| `rtdal transpile corpus` | Transpiles every available R source, updates matrices and creates the combined Go corpus. |

## Vector commands — planned

| Command | What it does |
| --- | --- |
| `rtdal vector info --input <dataset>` | Inspects layers, fields, geometry, extent and CRS. |
| `rtdal vector convert --input <dataset> --output <dataset>` | Converts vector datasets. |
| `rtdal vector bbox --input <dataset>` | Calculates a vector bounding box. |
| `rtdal vector reproject --input <dataset> --output <dataset> --target-crs <crs>` | Reprojects vector geometries. |
| `rtdal vector buffer --input <dataset> --output <dataset> --distance <value>` | Creates geometry buffers. |
| `rtdal vector simplify --input <dataset> --output <dataset> --tolerance <value>` | Simplifies geometries. |
| `rtdal vector centroid --input <dataset> --output <dataset>` | Calculates centroids. |
| `rtdal vector area --input <dataset> [--geodesic]` | Calculates feature areas. |
| `rtdal vector length --input <dataset> [--geodesic]` | Calculates feature lengths. |
| `rtdal vector distance --input <dataset> --other <dataset>` | Calculates geometry distances. |
| `rtdal vector intersect --input <dataset> --other <dataset> --output <dataset>` | Intersects layers. |
| `rtdal vector union --input <dataset> --output <dataset>` | Unions geometries. |
| `rtdal vector difference --input <dataset> --other <dataset> --output <dataset>` | Subtracts geometries. |
| `rtdal vector clip --input <dataset> --mask <dataset> --output <dataset>` | Clips a dataset. |
| `rtdal vector validate --input <dataset> [--repair]` | Validates or repairs geometry. |

## Raster commands — planned

| Command | What it does |
| --- | --- |
| `rtdal raster info --input <dataset>` | Inspects size, bands, type, extent, transform and CRS. |
| `rtdal raster convert --input <dataset> --output <dataset>` | Converts raster datasets. |
| `rtdal raster crop --input <dataset> --output <dataset> --bbox <xmin,ymin,xmax,ymax>` | Crops a raster. |
| `rtdal raster resample --input <dataset> --output <dataset> --resolution <x,y>` | Resamples a raster. |
| `rtdal raster reproject --input <dataset> --output <dataset> --target-crs <crs>` | Warps to a target CRS. |
| `rtdal raster mosaic --input <dataset...> --output <dataset>` | Mosaics raster inputs. |
| `rtdal raster calc --input <dataset...> --output <dataset> --expression <expr>` | Evaluates a raster expression. |
| `rtdal raster statistics --input <dataset>` | Calculates band statistics. |
| `rtdal raster tile --input <dataset> --output <directory> --zoom <min:max>` | Generates raster tiles. |

## Cube, STAC and GTFS — planned

| Command | What it does |
| --- | --- |
| `rtdal cube info|subset|aggregate|merge|warp ...` | Inspects, subsets, aggregates, merges or warps spatiotemporal data cubes. |
| `rtdal stac collections|search|assets|download ...` | Lists, searches, inspects or downloads STAC content. |
| `rtdal gtfs info|validate|subset|export ...` | Inspects, validates, subsets or exports GTFS feeds. |

## Geodesy and classification — planned

| Command | What it does |
| --- | --- |
| `rtdal geodesy distance|bearing|destination|polygon-area ...` | Performs geodesic measurement and destination calculations. |
| `rtdal classify equal|quantile|jenks|pretty --input <values> --classes <n>` | Builds common data-classification intervals. |

## Reproducible local build

```powershell
go run ./cmd/rtdal source fetch-all
go run ./cmd/rtdal transpile corpus
.\build-rtdal-onefile.bat
```

The source export deliberately excludes downloaded CRAN archives, Go caches,
generated corpus output and release binaries. `manifests/` documents conversion
coverage and omissions; `LICENSES/` and `THIRD_PARTY_NOTICES.md` preserve
upstream licence material.
