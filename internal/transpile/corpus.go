package transpile

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/compiler"
	"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/syntax"
)

type SourceFile struct {
	Package  string
	Path     string
	Relative string
}

type Result struct {
	Package             string
	Source              string
	Output              string
	Status              string
	Error               string
	Bytes               int
	TopLevelBlocks      int
	MatrixBlocks        int
	CompatibilityBlocks int
	RuntimeImport       bool
}

func Discover(root string) ([]SourceFile, error) {
	sourcesRoot := filepath.Join(root, "sources")
	packages, err := os.ReadDir(sourcesRoot)
	if err != nil {
		return nil, err
	}
	var files []SourceFile
	for _, pkg := range packages {
		if !pkg.IsDir() {
			continue
		}
		packageRoot := filepath.Join(sourcesRoot, pkg.Name())
		err := filepath.WalkDir(packageRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".R") {
				return nil
			}
			rel, err := filepath.Rel(packageRoot, path)
			if err != nil {
				return err
			}
			files = append(files, SourceFile{Package: pkg.Name(), Path: path, Relative: filepath.ToSlash(rel)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Package != files[j].Package {
			return strings.ToLower(files[i].Package) < strings.ToLower(files[j].Package)
		}
		return strings.ToLower(files[i].Relative) < strings.ToLower(files[j].Relative)
	})
	return files, nil
}

func Run(root string, workers int) ([]Result, error) {
	files, err := Discover(root)
	if err != nil {
		return nil, err
	}
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan SourceFile)
	results := make(chan Result, len(files))
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for file := range jobs {
				results <- transpileOne(root, file)
			}
		}()
	}
	go func() {
		for _, file := range files {
			jobs <- file
		}
		close(jobs)
		group.Wait()
		close(results)
	}()
	all := make([]Result, 0, len(files))
	for result := range results {
		all = append(all, result)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Package != all[j].Package {
			return strings.ToLower(all[i].Package) < strings.ToLower(all[j].Package)
		}
		return strings.ToLower(all[i].Source) < strings.ToLower(all[j].Source)
	})
	return all, nil
}

func transpileOne(root string, file SourceFile) (result Result) {
	result.Package = file.Package
	result.Source = filepath.ToSlash(filepath.Join("sources", file.Package, file.Relative))
	stem := strings.TrimSuffix(file.Relative, filepath.Ext(file.Relative))
	outputDir := filepath.Join(root, "generated", "r-files", file.Package, filepath.FromSlash(stem))
	outputPath := filepath.Join(outputDir, "main.go")
	result.Output = filepath.ToSlash(filepath.Join("generated", "r-files", file.Package, filepath.FromSlash(stem), "main.go"))
	data, err := os.ReadFile(file.Path)
	if err != nil {
		result.Status, result.Error = "read-failed", err.Error()
		return result
	}
	result.Bytes = len(data)
	defer func() {
		if recovered := recover(); recovered != nil {
			result.Status = "panic"
			result.Error = fmt.Sprint(recovered)
			_ = writeFailureStub(outputPath, file, data, result.Status, result.Error)
		}
	}()
	program, err := syntax.Parse(string(data))
	if err != nil {
		result.Status, result.Error = "parse-failed", err.Error()
		_ = writeFailureStub(outputPath, file, data, result.Status, result.Error)
		return result
	}
	result.TopLevelBlocks = len(program.Expressions)
	generated, err := compiler.GenerateMainWithOptions(program, string(data), compiler.GenerateOptions{AllowIRFallback: true, PreserveOriginal: true})
	if err != nil {
		result.Status, result.Error = "generate-failed", err.Error()
		_ = writeFailureStub(outputPath, file, data, result.Status, result.Error)
		return result
	}
	text := string(generated)
	result.MatrixBlocks = strings.Count(text, "matrix-lowered block")
	result.CompatibilityBlocks = strings.Count(text, "matrix lowering unavailable")
	result.RuntimeImport = strings.Contains(text, `"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/runtime"`)
	result.Status = "generated"
	if result.CompatibilityBlocks > 0 {
		result.Status = "generated-with-compatibility"
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Status, result.Error = "write-failed", err.Error()
		return result
	}
	if err := os.WriteFile(outputPath, generated, 0644); err != nil {
		result.Status, result.Error = "write-failed", err.Error()
	}
	return result
}

func writeFailureStub(path string, file SourceFile, source []byte, status, message string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var output strings.Builder
	output.WriteString("package main\n\n")
	fmt.Fprintf(&output, "// rtdal: %s while transpiling %s/%s\n", status, file.Package, file.Relative)
	for _, line := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		output.WriteString("// ERROR| ")
		output.WriteString(line)
		output.WriteByte('\n')
	}
	output.WriteString("// Original R source follows for the next repair pass.\n")
	for _, line := range strings.Split(strings.ReplaceAll(string(source), "\r\n", "\n"), "\n") {
		output.WriteString("// R| ")
		output.WriteString(line)
		output.WriteByte('\n')
	}
	output.WriteString("\nfunc main() {}\n")
	return os.WriteFile(path, []byte(output.String()), 0644)
}

func WriteManifest(path string, results []Result) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	header := []string{"package", "source", "output", "status", "error", "bytes", "top_level_blocks", "matrix_blocks", "compatibility_blocks", "runtime_import"}
	if err := writer.Write(header); err != nil {
		file.Close()
		return err
	}
	for _, result := range results {
		record := []string{result.Package, result.Source, result.Output, result.Status, result.Error, fmt.Sprint(result.Bytes), fmt.Sprint(result.TopLevelBlocks), fmt.Sprint(result.MatrixBlocks), fmt.Sprint(result.CompatibilityBlocks), fmt.Sprint(result.RuntimeImport)}
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

// WriteAggregate combines every R source unit into one real Go translation
// unit. Successful sources become executable loader functions over package-
// scoped runtime contexts; failures remain explicit no-op repair loaders with
// their original R source comments.
func WriteAggregate(root, path string, results []Result) error {
	files, err := Discover(root)
	if err != nil {
		return err
	}
	resultBySource := make(map[string]Result, len(results))
	for _, result := range results {
		resultBySource[result.Source] = result
	}
	var output strings.Builder
	output.WriteString("package generated\n\n")
	output.WriteString("// Code generated by RTDAL from the complete local R corpus. DO NOT EDIT.\n")
	output.WriteString("// Every loader is executable Go; repair loaders are explicitly marked.\n")
	output.WriteString("import (\n\t\"fmt\"\n\t\"github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library/toolchain/r2go/runtime\"\n)\n\n")
	output.WriteString("var _ = fmt.Errorf\n\n")
	type unit struct {
		pkg, source, status, function string
		library                       bool
	}
	units := make([]unit, 0, len(files))
	for index, file := range files {
		id := fmt.Sprintf("%06d", index+1)
		functionName := "loadRTDAL" + id
		constantPrefix := "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library" + id
		sourceKey := filepath.ToSlash(filepath.Join("sources", file.Package, file.Relative))
		result := resultBySource[sourceKey]
		data, readErr := os.ReadFile(file.Path)
		if readErr != nil {
			return readErr
		}
		program, parseErr := syntax.Parse(string(data))
		if parseErr == nil {
			loader, generateErr := compiler.GenerateMatrixLoader(program, string(data), functionName, constantPrefix, true)
			if generateErr == nil {
				body, splitErr := loaderDeclarations(loader)
				if splitErr != nil {
					return fmt.Errorf("merge %s: %w", sourceKey, splitErr)
				}
				output.WriteString(body)
				output.WriteString("\n\n")
			} else {
				writeAggregateOmittedLoader(&output, functionName, sourceKey, generateErr.Error())
			}
		} else {
			writeAggregateOmittedLoader(&output, functionName, sourceKey, parseErr.Error())
		}
		status := result.Status
		if status == "" {
			status = "unclassified"
		}
		library := strings.HasPrefix(filepath.ToSlash(file.Relative), "R/")
		units = append(units, unit{pkg: file.Package, source: sourceKey, status: status, function: functionName, library: library})
	}
	output.WriteString("type RTDALCorpusUnit struct {\n\tPackage string\n\tSource string\n\tStatus string\n\tLibrary bool\n\tLoad func(*runtime.Context)\n}\n\n")
	output.WriteString("var RTDALCorpus = []RTDALCorpusUnit{\n")
	for _, item := range units {
		fmt.Fprintf(&output, "\t{Package: %q, Source: %q, Status: %q, Library: %t, Load: %s},\n", item.pkg, item.source, item.status, item.library, item.function)
	}
	output.WriteString("}\n\n")
	output.WriteString("type RTDALLoadFailure struct { Source string; Error string }\n\n")
	output.WriteString("func LoadRTDALPackageWithReport(name string) (*runtime.Context, int, []RTDALLoadFailure) {\n")
	output.WriteString("\tctx := runtime.NewContext()\n\tloaded := 0\n\tfailures := []RTDALLoadFailure{}\n")
	output.WriteString("\tfor _, unit := range RTDALCorpus {\n\t\tif unit.Package != name || !unit.Library { continue }\n")
	output.WriteString("\t\tfunc() {\n\t\t\tdefer func() { if recovered := recover(); recovered != nil { failures = append(failures, RTDALLoadFailure{Source: unit.Source, Error: fmt.Sprint(recovered)}) } }()\n\t\t\tunit.Load(ctx)\n\t\t\tloaded++\n\t\t}()\n\t}\n")
	output.WriteString("\treturn ctx, loaded, failures\n}\n\n")
	output.WriteString("func LoadRTDALPackage(name string) *runtime.Context { ctx, _, _ := LoadRTDALPackageWithReport(name); return ctx }\n\n")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(output.String()), 0644)
}

func loaderDeclarations(source []byte) (string, error) {
	text := string(source)
	constant := strings.Index(text, "\nconst ")
	function := strings.Index(text, "\nfunc ")
	start := -1
	if constant >= 0 {
		start = constant + 1
	}
	if function >= 0 && (start < 0 || function+1 < start) {
		start = function + 1
	}
	if start < 0 {
		return "", fmt.Errorf("generated loader has no declarations")
	}
	return strings.TrimSpace(text[start:]), nil
}

func writeAggregateOmittedLoader(output *strings.Builder, functionName, sourceKey, message string) {
	fmt.Fprintf(output, "// rtdal: omitted unsupported source %s\n", sourceKey)
	for _, line := range strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n") {
		fmt.Fprintf(output, "// ERROR| %s\n", line)
	}
	fmt.Fprintf(output, "func %s(ctx *runtime.Context) { _ = ctx }\n\n", functionName)
}

var assignedFunction = regexp.MustCompile(`(?m)^\s*(` + "`?" + `[[:alnum:]_.%<>=:+*/-]+` + "`?" + `)\s*(?:<-|=)\s*function\s*\(`)

func WriteRemovedManifest(path, root string, results []Result) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"package", "source", "functions", "status", "error", "reason"}); err != nil {
		file.Close()
		return err
	}
	for _, result := range results {
		if result.Status != "parse-failed" && result.Status != "generate-failed" && result.Status != "panic" {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.Source)))
		matches := assignedFunction.FindAllSubmatch(data, -1)
		functions := make([]string, 0, len(matches))
		for _, match := range matches {
			if len(match) > 1 {
				functions = append(functions, strings.Trim(string(match[1]), "`"))
			}
		}
		reason := "unsupported R syntax excluded from executable corpus"
		if !strings.Contains(filepath.ToSlash(result.Source), "/R/") {
			reason = "non-library test, vignette, tool, or legacy source excluded"
		}
		if err := writer.Write([]string{result.Package, result.Source, strings.Join(functions, ";"), result.Status, result.Error, reason}); err != nil {
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
