package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"rtdal"
	"rtdal/generated"
	"rtdal/internal/command"
	"rtdal/internal/cran"
	"rtdal/internal/gui"
	"rtdal/internal/operations"
	"rtdal/internal/transpile"
)

var version = "development"

func main() {
	// A double-click provides no arguments. Open a persistent command prompt so
	// the user can read the command overview instead of seeing a flash-close.
	if len(os.Args) == 1 || (len(os.Args) == 2 && os.Args[1] == "--gui") {
		if err := gui.Show(os.Args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "rtdal:", err)
			os.Exit(1)
		}
		return
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rtdal:", err)
		os.Exit(1)
	}
}

func openHelpConsole() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	commandProcessor := os.Getenv("ComSpec")
	if commandProcessor == "" {
		commandProcessor = "cmd.exe"
	}
	command := fmt.Sprintf(`"%s" help`, executable)
	return exec.Command(commandProcessor, "/d", "/k", command).Start()
}
func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		filters := []string{}
		if len(args) > 1 {
			filters = args[1:]
		}
		fmt.Print(command.Help(filters...))
		return nil
	}
	switch args[0] {
	case "--license", "--licenses", "license", "licenses":
		fmt.Print(rtdal.LicenseReport())
		return nil
	case "--version", "version":
		fmt.Println("RTDAL", version)
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("missing command; use 'rtdal help %s'", args[0])
	}
	category, name := args[0], args[1]
	spec, ok := command.Find(category, name)
	if !ok {
		return fmt.Errorf("unknown command %q; use 'rtdal help'", category+" "+name)
	}
	if spec.State != command.Available {
		return fmt.Errorf("command %q is planned but not available in this build", category+" "+name)
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	switch category + " " + name {
	case "corpus list":
		totalCounts := map[string]int{}
		libraryCounts := map[string]int{}
		for _, unit := range generated.RTDALCorpus {
			totalCounts[unit.Package]++
			if unit.Library {
				libraryCounts[unit.Package]++
			}
		}
		packages := make([]string, 0, len(totalCounts))
		for packageName := range totalCounts {
			packages = append(packages, packageName)
		}
		sort.Strings(packages)
		for _, packageName := range packages {
			fmt.Printf("%-28s %d library / %d total source units\n", packageName, libraryCounts[packageName], totalCounts[packageName])
		}
		return nil
	case "corpus sources":
		packageName, err := optionValue(args[2:], "--package")
		if err != nil {
			return err
		}
		found := false
		for _, unit := range generated.RTDALCorpus {
			if unit.Package == packageName {
				fmt.Printf("%-30s %-30s %s\n", unit.Package, unit.Status, unit.Source)
				found = true
			}
		}
		if !found {
			return fmt.Errorf("embedded package %q not found", packageName)
		}
		return nil
	case "corpus load":
		packageName, err := optionValue(args[2:], "--package")
		if err != nil {
			return err
		}
		return loadCorpusPackage(packageName)
	case "source fetch":
		packages := args[2:]
		if len(packages) == 0 {
			packages = cran.CorePackages
		}
		return cran.Fetch(root, packages)
	case "source fetch-all":
		results, err := cran.FetchAll(root, 8)
		if err != nil {
			return err
		}
		return cran.WriteFetchReport(filepath.Join(root, "manifests", "source_fetch_matrix.csv"), results)
	case "source inventory":
		return writeInventory(root)
	case "source licenses":
		rows, err := cran.BuildMatrix(root)
		if err != nil {
			return err
		}
		return cran.WriteLicenseBundle(root, rows)
	case "transpile corpus":
		results, err := transpile.Run(root, 8)
		if err != nil {
			return err
		}
		if err := transpile.WriteManifest(filepath.Join(root, "manifests", "transpile_matrix.csv"), results); err != nil {
			return err
		}
		if err := transpile.WriteRemovedManifest(filepath.Join(root, "manifests", "removed_functions.csv"), root, results); err != nil {
			return err
		}
		return transpile.WriteAggregate(root, filepath.Join(root, "generated", "RTDAL_ALL_TRANSPILED.go"), results)
	}
	return operations.Run(category, name, args[2:])
}

func optionValue(args []string, name string) (string, error) {
	for index := 0; index < len(args); index++ {
		if args[index] == name {
			if index+1 >= len(args) || args[index+1] == "" {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[index+1], nil
		}
	}
	return "", fmt.Errorf("missing required option %s", name)
}

func loadCorpusPackage(packageName string) (err error) {
	found := false
	count := 0
	for _, unit := range generated.RTDALCorpus {
		if unit.Package == packageName && unit.Library {
			found = true
			count++
		}
	}
	if !found {
		return fmt.Errorf("embedded package %q not found", packageName)
	}
	_, loaded, failures := generated.LoadRTDALPackageWithReport(packageName)
	for _, failure := range failures {
		fmt.Fprintf(os.Stderr, "skipped %s: %s\n", failure.Source, failure.Error)
	}
	if loaded == 0 {
		return fmt.Errorf("no source unit from embedded package %q loaded successfully (%d skipped)", packageName, len(failures))
	}
	fmt.Printf("loaded %s into the Pure-Go runtime: %d/%d library source units (%d skipped)\n", packageName, loaded, count, len(failures))
	return nil
}

func writeInventory(root string) error {
	rows, err := cran.BuildMatrix(root)
	if err != nil {
		return err
	}
	if err := cran.WriteMatrix(filepath.Join(root, "manifests", "package_matrix.csv"), rows); err != nil {
		return err
	}
	return cran.WriteLicenseBundle(root, rows)
}
