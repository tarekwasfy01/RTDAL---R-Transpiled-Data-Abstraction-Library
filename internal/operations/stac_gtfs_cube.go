package operations

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func httpJSON(method, rawURL string, body any) (any, error) {
	var reader io.Reader
	if body != nil {
		data, e := json.Marshal(body)
		if e != nil {
			return nil, e
		}
		reader = bytes.NewReader(data)
	}
	req, e := http.NewRequest(method, rawURL, reader)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Accept", "application/geo+json, application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := http.Client{Timeout: 30 * time.Second}
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("HTTP %s: %s", resp.Status, strings.TrimSpace(string(snippet)))
	}
	var value any
	if e = json.NewDecoder(resp.Body).Decode(&value); e != nil {
		return nil, e
	}
	return value, nil
}
func runSTAC(name string, o options) error {
	switch name {
	case "collections":
		raw, e := o.value("--url")
		if e != nil {
			return e
		}
		v, e := httpJSON("GET", strings.TrimRight(raw, "/")+"/collections", nil)
		if e != nil {
			return e
		}
		return writeJSON(v)
	case "search":
		raw, e := o.value("--url")
		if e != nil {
			return e
		}
		query := map[string]any{}
		if b := o.optional("--bbox", ""); b != "" {
			v, e := numericList(b)
			if e != nil || len(v) != 4 {
				return fmt.Errorf("--bbox must contain four numbers")
			}
			query["bbox"] = v
		}
		if d := o.optional("--datetime", ""); d != "" {
			query["datetime"] = d
		}
		if q := o.optional("--query", ""); q != "" {
			var value any
			if json.Unmarshal([]byte(q), &value) != nil {
				return fmt.Errorf("--query must be JSON")
			}
			query["query"] = value
		}
		v, e := httpJSON("POST", strings.TrimRight(raw, "/")+"/search", query)
		if e != nil {
			return e
		}
		return writeJSON(v)
	case "assets":
		item, e := readJSONObject(o)
		if e != nil {
			return e
		}
		assets, _ := item["assets"].(map[string]any)
		return writeJSON(assets)
	case "download":
		item, e := readJSONObject(o)
		if e != nil {
			return e
		}
		dir, e := o.value("--output")
		if e != nil {
			return e
		}
		assets, _ := item["assets"].(map[string]any)
		selected := o.optional("--asset", "")
		if e = os.MkdirAll(dir, 0755); e != nil {
			return e
		}
		count := 0
		for key, raw := range assets {
			if selected != "" && key != selected {
				continue
			}
			meta, _ := raw.(map[string]any)
			href, _ := meta["href"].(string)
			if href == "" {
				continue
			}
			if e = downloadFile(href, filepath.Join(dir, assetFilename(key, href))); e != nil {
				return fmt.Errorf("asset %s: %w", key, e)
			}
			count++
		}
		if count == 0 {
			return fmt.Errorf("no matching downloadable assets")
		}
		return writeJSON(map[string]any{"downloaded": count, "directory": dir})
	}
	return fmt.Errorf("unsupported STAC command %q", name)
}
func readJSONObject(o options) (map[string]any, error) {
	path, e := o.value("--input")
	if e != nil {
		return nil, e
	}
	data, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var v map[string]any
	if e = json.Unmarshal(data, &v); e != nil {
		return nil, e
	}
	return v, nil
}
func assetFilename(key, href string) string {
	u, e := url.Parse(href)
	if e == nil && filepath.Base(u.Path) != "." && filepath.Base(u.Path) != "/" {
		return filepath.Base(u.Path)
	}
	return key
}
func downloadFile(rawURL, path string) error {
	resp, e := http.Get(rawURL)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = io.Copy(f, resp.Body)
	return e
}

type gtfsTable struct {
	Name   string
	Header []string
	Rows   [][]string
}

func loadGTFS(path string) ([]gtfsTable, error) {
	zr, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	defer zr.Close()
	var tables []gtfsTable
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".txt") {
			continue
		}
		rc, e := f.Open()
		if e != nil {
			return nil, e
		}
		records, e := csv.NewReader(rc).ReadAll()
		rc.Close()
		if e != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, e)
		}
		if len(records) == 0 {
			continue
		}
		tables = append(tables, gtfsTable{f.Name, records[0], records[1:]})
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	return tables, nil
}
func runGTFS(name string, o options) error {
	in, e := o.value("--input")
	if e != nil {
		return e
	}
	tables, e := loadGTFS(in)
	if e != nil {
		return e
	}
	switch name {
	case "info":
		summary := map[string]any{}
		for _, t := range tables {
			summary[t.Name] = map[string]any{"rows": len(t.Rows), "columns": t.Header}
		}
		return writeJSON(map[string]any{"tables": summary, "table_count": len(tables)})
	case "validate":
		issues := validateGTFS(tables)
		return writeJSON(map[string]any{"valid": len(issues) == 0, "issues": issues, "tables": len(tables)})
	case "export":
		dir, e := o.value("--output")
		if e != nil {
			return e
		}
		if e = os.MkdirAll(dir, 0755); e != nil {
			return e
		}
		for _, t := range tables {
			if e = writeCSV(filepath.Join(dir, filepath.Base(t.Name)), t); e != nil {
				return e
			}
		}
		return nil
	case "subset":
		route, e := o.value("--route")
		if e != nil {
			return e
		}
		out, e := o.value("--output")
		if e != nil {
			return e
		}
		tables = subsetGTFS(tables, route)
		return writeGTFSZip(out, tables)
	}
	return fmt.Errorf("unsupported GTFS command %q", name)
}
func validateGTFS(t []gtfsTable) []string {
	present := map[string]bool{}
	for _, x := range t {
		present[strings.ToLower(filepath.Base(x.Name))] = true
	}
	var issues []string
	for _, required := range []string{"agency.txt", "stops.txt", "routes.txt", "trips.txt", "stop_times.txt"} {
		if !present[required] {
			issues = append(issues, "missing "+required)
		}
	}
	for _, x := range t {
		seen := map[string]bool{}
		for _, h := range x.Header {
			if h == "" {
				issues = append(issues, x.Name+": empty header")
			}
			if seen[h] {
				issues = append(issues, x.Name+": duplicate header "+h)
			}
			seen[h] = true
		}
		for i, row := range x.Rows {
			if len(row) != len(x.Header) {
				issues = append(issues, fmt.Sprintf("%s row %d has %d columns, expected %d", x.Name, i+2, len(row), len(x.Header)))
			}
		}
	}
	return issues
}
func subsetGTFS(t []gtfsTable, route string) []gtfsTable {
	tripIDs := map[string]bool{}
	for i := range t {
		if strings.EqualFold(filepath.Base(t[i].Name), "trips.txt") {
			ri, ti := columnIndex(t[i].Header, "route_id"), columnIndex(t[i].Header, "trip_id")
			var rows [][]string
			for _, r := range t[i].Rows {
				if ri >= 0 && ti >= 0 && r[ri] == route {
					rows = append(rows, r)
					tripIDs[r[ti]] = true
				}
			}
			t[i].Rows = rows
		}
	}
	for i := range t {
		base := strings.ToLower(filepath.Base(t[i].Name))
		if base == "routes.txt" {
			idx := columnIndex(t[i].Header, "route_id")
			t[i].Rows = filterRows(t[i].Rows, idx, func(v string) bool { return v == route })
		}
		if base == "stop_times.txt" {
			idx := columnIndex(t[i].Header, "trip_id")
			t[i].Rows = filterRows(t[i].Rows, idx, func(v string) bool { return tripIDs[v] })
		}
	}
	return t
}
func columnIndex(h []string, name string) int {
	for i, v := range h {
		if v == name {
			return i
		}
	}
	return -1
}
func filterRows(rows [][]string, idx int, keep func(string) bool) [][]string {
	if idx < 0 {
		return rows
	}
	var out [][]string
	for _, r := range rows {
		if idx < len(r) && keep(r[idx]) {
			out = append(out, r)
		}
	}
	return out
}
func writeCSV(path string, t gtfsTable) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	w := csv.NewWriter(f)
	e = w.Write(t.Header)
	if e == nil {
		e = w.WriteAll(t.Rows)
	}
	w.Flush()
	if e == nil {
		e = w.Error()
	}
	return e
}
func writeGTFSZip(path string, t []gtfsTable) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	z := zip.NewWriter(f)
	for _, table := range t {
		w, e := z.Create(filepath.ToSlash(table.Name))
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
		cw := csv.NewWriter(w)
		if e = cw.Write(table.Header); e == nil {
			e = cw.WriteAll(table.Rows)
		}
		cw.Flush()
		if e == nil {
			e = cw.Error()
		}
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
	}
	if e = z.Close(); e != nil {
		f.Close()
		return e
	}
	return f.Close()
}

type cube struct {
	Dimensions []string       `json:"dimensions"`
	Shape      []int          `json:"shape"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Data       []float64      `json:"data"`
}

func loadCube(path string) (cube, error) {
	data, e := os.ReadFile(path)
	if e != nil {
		return cube{}, e
	}
	var c cube
	e = json.Unmarshal(data, &c)
	if e != nil {
		return c, e
	}
	size := 1
	for _, n := range c.Shape {
		if n <= 0 {
			return c, fmt.Errorf("invalid cube shape")
		}
		size *= n
	}
	if len(c.Dimensions) != len(c.Shape) || len(c.Data) != size {
		return c, fmt.Errorf("cube dimensions, shape and data length do not match")
	}
	return c, nil
}
func saveCube(path string, c cube) error {
	data, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func runCube(name string, o options) error {
	paths := o.values("--input")
	if len(paths) == 0 {
		return fmt.Errorf("missing required option --input")
	}
	cubes := make([]cube, len(paths))
	var e error
	for i, p := range paths {
		cubes[i], e = loadCube(p)
		if e != nil {
			return e
		}
	}
	c := cubes[0]
	switch name {
	case "info":
		return writeJSON(map[string]any{"dimensions": c.Dimensions, "shape": c.Shape, "attributes": c.Attributes, "cells": len(c.Data)})
	case "subset":
		sel := o.optional("--select", "")
		if sel != "" {
			parts := strings.Split(sel, ":")
			if len(parts) != 2 {
				return fmt.Errorf("--select supports start:end over flattened cells")
			}
			a, _ := strconv.Atoi(parts[0])
			b, _ := strconv.Atoi(parts[1])
			if a < 0 || b < a || b > len(c.Data) {
				return fmt.Errorf("invalid subset range")
			}
			c.Data = append([]float64(nil), c.Data[a:b]...)
			c.Dimensions = []string{"cell"}
			c.Shape = []int{len(c.Data)}
		}
		out, e := o.value("--output")
		if e != nil {
			return e
		}
		return saveCube(out, c)
	case "aggregate":
		fn, e := o.value("--function")
		if e != nil {
			return e
		}
		var v float64
		switch fn {
		case "sum", "mean":
			for _, x := range c.Data {
				v += x
			}
			if fn == "mean" {
				v /= float64(len(c.Data))
			}
		case "min":
			v = c.Data[0]
			for _, x := range c.Data {
				if x < v {
					v = x
				}
			}
		case "max":
			v = c.Data[0]
			for _, x := range c.Data {
				if x > v {
					v = x
				}
			}
		default:
			return fmt.Errorf("aggregate function supports sum, mean, min, max")
		}
		c.Dimensions = []string{"aggregate"}
		c.Shape = []int{1}
		c.Data = []float64{v}
		out, e := o.value("--output")
		if e != nil {
			return e
		}
		return saveCube(out, c)
	case "merge":
		for _, other := range cubes[1:] {
			c.Data = append(c.Data, other.Data...)
		}
		c.Dimensions = []string{"cell"}
		c.Shape = []int{len(c.Data)}
		out, e := o.value("--output")
		if e != nil {
			return e
		}
		return saveCube(out, c)
	case "warp":
		target, e := o.value("--target")
		if e != nil {
			return e
		}
		shapeRaw := strings.TrimPrefix(target, "shape=")
		vals, e := numericList(shapeRaw)
		if e != nil {
			return fmt.Errorf("--target must be shape=n,m,...")
		}
		shape := make([]int, len(vals))
		size := 1
		for i, v := range vals {
			shape[i] = int(v)
			size *= shape[i]
		}
		data := make([]float64, size)
		for i := range data {
			data[i] = c.Data[i*len(c.Data)/size]
		}
		c.Shape = shape
		c.Dimensions = make([]string, len(shape))
		for i := range shape {
			c.Dimensions[i] = fmt.Sprintf("dim%d", i+1)
		}
		c.Data = data
		out, e := o.value("--output")
		if e != nil {
			return e
		}
		return saveCube(out, c)
	}
	return fmt.Errorf("unsupported cube command %q", name)
}
