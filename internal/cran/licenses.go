package cran

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var licenseFileNames = map[string]bool{
	"license":   true,
	"licence":   true,
	"copying":   true,
	"copyright": true,
	"notice":    true,
}

// WriteLicenseBundle preserves the exact CRAN licence declaration and every
// package-provided licence/notice file. Transpiled components keep their own
// provenance instead of being silently relicensed as one homogeneous work.
func WriteLicenseBundle(root string, rows []MatrixRow) error {
	licensesRoot := filepath.Join(root, "LICENSES")
	if err := os.MkdirAll(licensesRoot, 0755); err != nil {
		return err
	}
	var notice strings.Builder
	notice.WriteString("# RTDAL third-party source provenance\n\n")
	notice.WriteString("RTDAL uses the following R packages as source and behavioural references. ")
	notice.WriteString("Each component retains its declared upstream licence.\n\n")
	for _, row := range rows {
		if !row.SourceAvailable {
			continue
		}
		pkg := row.Package
		pkgLicenseDir := filepath.Join(licensesRoot, pkg.Name)
		if err := os.MkdirAll(pkgLicenseDir, 0755); err != nil {
			return err
		}
		declaration := fmt.Sprintf("Package: %s\nVersion: %s\nSource: https://cran.r-project.org/package=%s\nDeclared-License: %s\nTranspile-Class: %s\n", pkg.Name, pkg.Version, pkg.Name, pkg.License, row.TranspileClass)
		if err := os.WriteFile(filepath.Join(pkgLicenseDir, "LICENSE_DECLARATION.txt"), []byte(declaration), 0644); err != nil {
			return err
		}
		files, err := findLicenseFiles(pkg.SourceDir)
		if err != nil {
			return err
		}
		for _, source := range files {
			data, err := os.ReadFile(source)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(pkg.SourceDir, source)
			if err != nil {
				return err
			}
			targetName := strings.ReplaceAll(filepath.ToSlash(rel), "/", "__")
			if err := os.WriteFile(filepath.Join(pkgLicenseDir, targetName), data, 0644); err != nil {
				return err
			}
		}
		fmt.Fprintf(&notice, "## %s %s\n\n- Source: https://cran.r-project.org/package=%s\n- Declared licence: `%s`\n- Classification: `%s`\n- Preserved licence files: %d\n\n", pkg.Name, pkg.Version, pkg.Name, pkg.License, row.TranspileClass, len(files))
	}
	return os.WriteFile(filepath.Join(root, "THIRD_PARTY_NOTICES.md"), []byte(notice.String()), 0644)
}

func findLicenseFiles(sourceRoot string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			rel, _ := filepath.Rel(sourceRoot, path)
			if rel != "." && strings.Count(filepath.ToSlash(rel), "/") >= 2 {
				return filepath.SkipDir
			}
			return nil
		}
		base := strings.ToLower(entry.Name())
		base = strings.TrimSuffix(base, filepath.Ext(base))
		if licenseFileNames[base] {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}
