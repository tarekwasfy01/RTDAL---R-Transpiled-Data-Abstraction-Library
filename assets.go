package rtdal

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed THIRD_PARTY_NOTICES.md LICENSES
var legalFiles embed.FS

// LicenseReport returns the provenance overview and all embedded upstream
// licence files. The same bytes are carried inside the single RTDAL executable.
func LicenseReport() string {
	var output strings.Builder
	if notice, err := legalFiles.ReadFile("THIRD_PARTY_NOTICES.md"); err == nil {
		output.Write(notice)
		output.WriteString("\n")
	}
	var paths []string
	_ = fs.WalkDir(legalFiles, "LICENSES", func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	for _, path := range paths {
		data, err := legalFiles.ReadFile(path)
		if err != nil {
			continue
		}
		fmt.Fprintf(&output, "\n================================================================================\nFILE: %s\n================================================================================\n", path)
		output.Write(data)
		if len(data) == 0 || data[len(data)-1] != '\n' {
			output.WriteByte('\n')
		}
	}
	return output.String()
}
