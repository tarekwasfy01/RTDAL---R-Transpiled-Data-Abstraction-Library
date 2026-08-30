package operations

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type options map[string][]string

func parseOptions(args []string) (options, error) {
	o := options{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			return nil, fmt.Errorf("unexpected argument %q", args[i])
		}
		key := args[i]
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			o[key] = append(o[key], args[i+1])
			i++
		} else {
			o[key] = append(o[key], "true")
		}
	}
	return o, nil
}
func (o options) value(name string) (string, error) {
	v := o[name]
	if len(v) == 0 || v[0] == "" || v[0] == "true" {
		return "", fmt.Errorf("missing required option %s", name)
	}
	return v[0], nil
}
func (o options) optional(name, fallback string) string {
	if v := o[name]; len(v) != 0 && v[0] != "true" {
		return v[0]
	}
	return fallback
}
func (o options) flag(name string) bool { return len(o[name]) != 0 }
func (o options) values(name string) []string {
	var out []string
	for _, raw := range o[name] {
		for _, item := range strings.Split(raw, ",") {
			if strings.TrimSpace(item) != "" {
				out = append(out, strings.TrimSpace(item))
			}
		}
	}
	return out
}
func parseFloat(value, label string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", label, value, err)
	}
	return v, nil
}
func writeJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func Run(category, name string, args []string) error {
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	switch category {
	case "classify":
		return runClassify(name, o)
	case "geodesy":
		return runGeodesy(name, o)
	case "vector":
		return runVector(name, o)
	case "raster":
		return runRaster(name, o)
	case "stac":
		return runSTAC(name, o)
	case "gtfs":
		return runGTFS(name, o)
	case "cube":
		return runCube(name, o)
	}
	return fmt.Errorf("no handler for %q", category+" "+name)
}
