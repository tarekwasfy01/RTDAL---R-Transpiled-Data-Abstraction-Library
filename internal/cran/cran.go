package cran

import (
	"archive/tar"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var CorePackages = []string{"abind", "stars", "rstac", "gtfsio"}

type Package struct {
	Name             string
	Version          string
	License          string
	NeedsCompilation string
	Depends          []string
	MetadataDir      string
	SourceDir        string
}

type MatrixRow struct {
	Package         Package
	SourceAvailable bool
	RFiles          int
	RLines          int
	NativeFiles     int
	NativeCallSites int
	LicenseClass    string
	TranspileClass  string
}

type FetchResult struct {
	Package string
	Version string
	Status  string
	Detail  string
}

func Discover(root string) ([]Package, error) {
	metadataRoot := filepath.Join(root, "R-Packages")
	entries, err := os.ReadDir(metadataRoot)
	if err != nil {
		return nil, fmt.Errorf("read package inventory: %w", err)
	}
	packages := make([]Package, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(metadataRoot, entry.Name())
		fields, err := readDCF(filepath.Join(dir, "DESCRIPTION"))
		if err != nil {
			continue
		}
		name, version := fields["Package"], fields["Version"]
		if name == "" || version == "" {
			continue
		}
		packages = append(packages, Package{
			Name:             name,
			Version:          version,
			License:          fields["License"],
			NeedsCompilation: fields["NeedsCompilation"],
			Depends:          dependencies(fields),
			MetadataDir:      dir,
			SourceDir:        filepath.Join(root, "sources", name),
		})
	}
	sort.Slice(packages, func(i, j int) bool { return strings.ToLower(packages[i].Name) < strings.ToLower(packages[j].Name) })
	return packages, nil
}

func readDCF(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	current := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if current != "" {
				fields[current] += " " + strings.TrimSpace(line)
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			current = ""
			continue
		}
		current = strings.TrimSpace(key)
		fields[current] = strings.TrimSpace(value)
	}
	return fields, nil
}

var versionConstraint = regexp.MustCompile(`\s*\([^)]*\)`)

func dependencies(fields map[string]string) []string {
	seen := map[string]bool{}
	var result []string
	for _, key := range []string{"Depends", "Imports", "LinkingTo"} {
		for _, item := range strings.Split(fields[key], ",") {
			name := strings.TrimSpace(versionConstraint.ReplaceAllString(item, ""))
			if name == "" || name == "R" || seen[name] {
				continue
			}
			seen[name] = true
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func Fetch(root string, selected []string) error {
	packages, err := Discover(root)
	if err != nil {
		return err
	}
	byName := make(map[string]Package, len(packages))
	for _, pkg := range packages {
		byName[strings.ToLower(pkg.Name)] = pkg
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	for _, name := range selected {
		pkg, ok := byName[strings.ToLower(name)]
		if !ok {
			return fmt.Errorf("package %q is absent from R-Packages", name)
		}
		if err := fetchOne(client, root, pkg); err != nil {
			return err
		}
		fmt.Printf("FETCH PASS package=%s version=%s\n", pkg.Name, pkg.Version)
	}
	return nil
}

// FetchAll downloads the complete metadata inventory with bounded concurrency.
// Individual unavailable base/recommended packages are recorded, not allowed to
// abort the remaining CRAN corpus.
func FetchAll(root string, workers int) ([]FetchResult, error) {
	packages, err := Discover(root)
	if err != nil {
		return nil, err
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan Package)
	results := make(chan FetchResult, len(packages))
	client := &http.Client{Timeout: 2 * time.Minute}
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for pkg := range jobs {
				status := "fetched"
				if sourceReady(pkg) {
					status = "cached"
				}
				err := fetchOne(client, root, pkg)
				result := FetchResult{Package: pkg.Name, Version: pkg.Version, Status: status}
				if err != nil {
					result.Status = "unavailable"
					result.Detail = err.Error()
				}
				results <- result
			}
		}()
	}
	go func() {
		for _, pkg := range packages {
			jobs <- pkg
		}
		close(jobs)
		group.Wait()
		close(results)
	}()
	all := make([]FetchResult, 0, len(packages))
	for result := range results {
		all = append(all, result)
	}
	sort.Slice(all, func(i, j int) bool { return strings.ToLower(all[i].Package) < strings.ToLower(all[j].Package) })
	return all, nil
}

func WriteFetchReport(path string, results []FetchResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"package", "version", "status", "detail"}); err != nil {
		file.Close()
		return err
	}
	for _, result := range results {
		if err := writer.Write([]string{result.Package, result.Version, result.Status, result.Detail}); err != nil {
			file.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func fetchOne(client *http.Client, root string, pkg Package) error {
	if sourceReady(pkg) {
		return nil
	}
	file := fmt.Sprintf("%s_%s.tar.gz", pkg.Name, pkg.Version)
	urls := []string{
		"https://cran.r-project.org/src/contrib/" + file,
		"https://cran.r-project.org/src/contrib/Archive/" + pkg.Name + "/" + file,
	}
	var lastStatus string
	for _, url := range urls {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "RTDAL source inventory/0.1")
		response, err := client.Do(req)
		if err != nil {
			lastStatus = err.Error()
			continue
		}
		if response.StatusCode != http.StatusOK {
			lastStatus = response.Status
			response.Body.Close()
			continue
		}
		err = extractTarGZ(response.Body, filepath.Join(root, "sources"))
		closeErr := response.Body.Close()
		if err != nil {
			return fmt.Errorf("extract %s: %w", pkg.Name, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s source: %w", pkg.Name, closeErr)
		}
		return nil
	}
	return fmt.Errorf("download %s %s: %s", pkg.Name, pkg.Version, lastStatus)
}

func sourceReady(pkg Package) bool {
	fields, err := readDCF(filepath.Join(pkg.SourceDir, "DESCRIPTION"))
	return err == nil && fields["Package"] == pkg.Name && fields["Version"] == pkg.Version
}

func extractTarGZ(input io.Reader, destination string) error {
	gz, err := gzip.NewReader(input)
	if err != nil {
		return err
	}
	defer gz.Close()
	archive := tar.NewReader(gz)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(filepath.FromSlash(header.Name))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		target := filepath.Join(destination, clean)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, archive)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}

var nativeCall = regexp.MustCompile(`\.(Call|C|External|Fortran)\s*\(`)

func BuildMatrix(root string) ([]MatrixRow, error) {
	packages, err := Discover(root)
	if err != nil {
		return nil, err
	}
	rows := make([]MatrixRow, 0, len(packages))
	for _, pkg := range packages {
		row := MatrixRow{Package: pkg, LicenseClass: licenseClass(pkg.License)}
		if info, err := os.Stat(pkg.SourceDir); err == nil && info.IsDir() {
			row.SourceAvailable = true
			err = filepath.WalkDir(pkg.SourceDir, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil || entry.IsDir() {
					return walkErr
				}
				ext := strings.ToLower(filepath.Ext(path))
				switch ext {
				case ".r":
					row.RFiles++
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					row.RLines += strings.Count(string(data), "\n") + 1
					row.NativeCallSites += len(nativeCall.FindAll(data, -1))
				case ".c", ".cc", ".cpp", ".cxx", ".f", ".f90":
					row.NativeFiles++
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		row.TranspileClass = classify(row)
		rows = append(rows, row)
	}
	return rows, nil
}

func classify(row MatrixRow) string {
	if !row.SourceAvailable {
		return "source-missing"
	}
	if row.NativeFiles == 0 && row.NativeCallSites == 0 && strings.EqualFold(row.Package.NeedsCompilation, "no") {
		return "pure-r-candidate"
	}
	if row.RFiles > 0 {
		return "mixed-r-native-boundary"
	}
	return "native-only-or-data"
}

func licenseClass(license string) string {
	lower := strings.ToLower(license)
	if strings.Contains(lower, "mit") || strings.Contains(lower, "apache") || strings.Contains(lower, "bsd") || strings.Contains(lower, "artistic") {
		return "permissive-or-weak"
	}
	if strings.Contains(lower, "gpl") {
		return "copyleft"
	}
	return "review-required"
}

func WriteMatrix(path string, rows []MatrixRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"package", "version", "license", "license_class", "needs_compilation", "source_available", "r_files", "r_lines", "native_files", "r_native_call_sites", "transpile_class", "dependencies"}); err != nil {
		file.Close()
		return err
	}
	for _, row := range rows {
		record := []string{row.Package.Name, row.Package.Version, row.Package.License, row.LicenseClass, row.Package.NeedsCompilation, fmt.Sprint(row.SourceAvailable), fmt.Sprint(row.RFiles), fmt.Sprint(row.RLines), fmt.Sprint(row.NativeFiles), fmt.Sprint(row.NativeCallSites), row.TranspileClass, strings.Join(row.Package.Depends, ";")}
		if err := writer.Write(record); err != nil {
			file.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
