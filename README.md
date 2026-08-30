# RTDAL

RTDAL (R-Transpiled Data Abstraction Library) turns selected, licence-compatible
R geospatial packages into reviewed Pure-Go building blocks. It is also a real
package corpus for hardening the R2Go transpiler.

The checked-in `R-Packages` directory is the version and licence inventory. The
actual CRAN source archives are fetched reproducibly into the ignored `sources`
directory.

## First package wave

| Package | Role | Licence | Initial treatment |
| --- | --- | --- | --- |
| `abind` | multidimensional array binding | MIT | Pure-Go foundation |
| `stars` | spatiotemporal array/data-cube semantics | Apache | transpile R algorithms; replace `sf` boundaries |
| `rstac` | STAC data access | MIT | transpile data/request logic; provide Go HTTP/JSON adapters |
| `gtfsio` | transport data I/O | MIT | transpile schema and table logic; provide Go file adapters |

Packages such as `sf`, `terra`, `raster`, `s2`, `lwgeom`, and `geosphere` remain
in the compatibility matrix. Their R wrappers can be analysed, but native
C/C++ algorithms and GPL boundaries are not presented as automatically
translated Pure-Go code.

## CLI

```powershell
RTDAL.exe help
RTDAL.exe help raster
RTDAL.exe corpus list
RTDAL.exe corpus sources --package abind
RTDAL.exe corpus load --package abind
RTDAL.exe --license
```

The command grammar is consistently `rtdal <category> <command> [options]`.
`help` groups commands by topic and labels unfinished GIS commands as `planned`.
The `corpus` commands operate on the transpiled package loaders embedded in the
one-file executable. Only package library sources below `R/` are executed;
tests, vignettes, tools, and legacy sources remain in the inventory but are not
loaded as library code. A failing library source is reported and skipped so it
does not prevent the remaining source units of that package from loading.

Development commands reproduce the source and transpilation matrices:

```powershell
RTDAL.exe source fetch-all
RTDAL.exe source inventory
RTDAL.exe source licenses
RTDAL.exe transpile corpus
```

`source fetch-all` downloads the exact versions named by the local DESCRIPTION
files. `source inventory` writes `manifests/package_matrix.csv` with source,
licence, R-code, native-code, and native-call counts. `transpile corpus` writes
the per-file matrix, the explicit removal manifest, and the single combined Go
translation unit.

Every fetched component receives an exact declaration and preserved upstream
licence files under `LICENSES/<package>`. `THIRD_PARTY_NOTICES.md` is generated
from the same matrix, so translated source never loses its origin or licence.

## Build

```powershell
.\build-rtdal-onefile.bat
```

The build uses `CGO_ENABLED=0` and produces `dist/RTDAL.exe`. The executable
contains the Pure-Go compatibility runtime, the generated corpus, CLI help, and
all collected licence texts. Native `.C`, `.Call`, and `.External` algorithms
are not silently claimed as Pure Go; unsupported runtime paths remain explicit.
