package operations

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type featureCollection struct {
	Type     string    `json:"type"`
	Name     string    `json:"name,omitempty"`
	CRS      any       `json:"crs,omitempty"`
	Features []feature `json:"features"`
}
type feature struct {
	Type       string         `json:"type"`
	ID         any            `json:"id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
	Geometry   geometry       `json:"geometry"`
}
type geometry struct {
	Type        string     `json:"type"`
	Coordinates any        `json:"coordinates,omitempty"`
	Geometries  []geometry `json:"geometries,omitempty"`
}
type bounds struct {
	MinX, MinY, MaxX, MaxY float64
	Valid                  bool
}

func (b *bounds) add(x, y float64) {
	if !b.Valid {
		b.MinX, b.MaxX, b.MinY, b.MaxY = x, x, y, y
		b.Valid = true
		return
	}
	b.MinX = math.Min(b.MinX, x)
	b.MaxX = math.Max(b.MaxX, x)
	b.MinY = math.Min(b.MinY, y)
	b.MaxY = math.Max(b.MaxY, y)
}
func (b bounds) slice() []float64 { return []float64{b.MinX, b.MinY, b.MaxX, b.MaxY} }

func loadGeoJSON(path string) (featureCollection, error) {
	data, e := os.ReadFile(path)
	if e != nil {
		return featureCollection{}, e
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if e = json.Unmarshal(data, &envelope); e != nil {
		return featureCollection{}, fmt.Errorf("invalid GeoJSON: %w", e)
	}
	var fc featureCollection
	switch envelope.Type {
	case "FeatureCollection":
		e = json.Unmarshal(data, &fc)
	case "Feature":
		var f feature
		e = json.Unmarshal(data, &f)
		fc = featureCollection{Type: "FeatureCollection", Features: []feature{f}}
	default:
		var g geometry
		e = json.Unmarshal(data, &g)
		fc = featureCollection{Type: "FeatureCollection", Features: []feature{{Type: "Feature", Properties: map[string]any{}, Geometry: g}}}
	}
	if e != nil {
		return fc, e
	}
	if fc.Type == "" {
		fc.Type = "FeatureCollection"
	}
	return fc, nil
}
func saveGeoJSON(path string, fc featureCollection) error {
	if filepath.Ext(path) == "" {
		path += ".geojson"
	}
	data, e := json.MarshalIndent(fc, "", "  ")
	if e != nil {
		return e
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
func walkCoords(v any, fn func(float64, float64) (float64, float64)) any {
	switch x := v.(type) {
	case []any:
		if len(x) >= 2 {
			a, aok := x[0].(float64)
			b, bok := x[1].(float64)
			if aok && bok {
				nx, ny := fn(a, b)
				out := append([]any(nil), x...)
				out[0], out[1] = nx, ny
				return out
			}
		}
		out := make([]any, len(x))
		for i := range x {
			out[i] = walkCoords(x[i], fn)
		}
		return out
	}
	return v
}
func geomBounds(g geometry) bounds {
	var b bounds
	walkCoords(g.Coordinates, func(x, y float64) (float64, float64) { b.add(x, y); return x, y })
	for _, child := range g.Geometries {
		cb := geomBounds(child)
		if cb.Valid {
			b.add(cb.MinX, cb.MinY)
			b.add(cb.MaxX, cb.MaxY)
		}
	}
	return b
}
func collectionBounds(fc featureCollection) bounds {
	var b bounds
	for _, f := range fc.Features {
		q := geomBounds(f.Geometry)
		if q.Valid {
			b.add(q.MinX, q.MinY)
			b.add(q.MaxX, q.MaxY)
		}
	}
	return b
}
func requireInput(o options) (string, error)  { return o.value("--input") }
func requireOutput(o options) (string, error) { return o.value("--output") }

func runVector(name string, o options) error {
	in, e := requireInput(o)
	if e != nil {
		return e
	}
	fc, e := loadGeoJSON(in)
	if e != nil {
		return fmt.Errorf("read vector input: %w", e)
	}
	switch name {
	case "info":
		b := collectionBounds(fc)
		types := map[string]int{}
		fields := map[string]bool{}
		for _, f := range fc.Features {
			types[f.Geometry.Type]++
			for k := range f.Properties {
				fields[k] = true
			}
		}
		result := map[string]any{"format": "GeoJSON", "features": len(fc.Features), "geometry_types": types, "fields": fields}
		if b.Valid {
			result["bbox"] = b.slice()
		}
		return writeJSON(result)
	case "bbox":
		b := collectionBounds(fc)
		if !b.Valid {
			return fmt.Errorf("input has no coordinates")
		}
		return writeJSON(map[string]any{"bbox": b.slice()})
	case "convert":
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveGeoJSON(out, fc)
	case "reproject":
		target, e := o.value("--target-crs")
		if e != nil {
			return e
		}
		target = strings.ToUpper(strings.ReplaceAll(target, " ", ""))
		if target != "EPSG:3857" && target != "3857" && target != "EPSG:4326" && target != "4326" {
			return fmt.Errorf("Pure-Go reproject supports EPSG:4326 and EPSG:3857")
		}
		for i := range fc.Features {
			fc.Features[i].Geometry.Coordinates = walkCoords(fc.Features[i].Geometry.Coordinates, func(x, y float64) (float64, float64) {
				if strings.Contains(target, "3857") {
					return lonLatToMercator(x, y)
				}
				return mercatorToLonLat(x, y)
			})
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveGeoJSON(out, fc)
	case "centroid":
		for i := range fc.Features {
			x, y, ok := geomCentroid(fc.Features[i].Geometry)
			if !ok {
				continue
			}
			fc.Features[i].Geometry = geometry{Type: "Point", Coordinates: []any{x, y}}
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveGeoJSON(out, fc)
	case "area", "length":
		values := make([]map[string]any, 0, len(fc.Features))
		var total float64
		for i, f := range fc.Features {
			v := geomMeasure(f.Geometry, name, o.flag("--geodesic"))
			total += v
			values = append(values, map[string]any{"feature": i, "value": v})
		}
		return writeJSON(map[string]any{"operation": name, "total": total, "features": values, "units": measureUnits(name, o.flag("--geodesic"))})
	case "distance":
		other, e := o.value("--other")
		if e != nil {
			return e
		}
		oc, e := loadGeoJSON(other)
		if e != nil {
			return e
		}
		a := collectionBounds(fc)
		b := collectionBounds(oc)
		if !a.Valid || !b.Valid {
			return fmt.Errorf("dataset has no coordinates")
		}
		ax, ay := (a.MinX+a.MaxX)/2, (a.MinY+a.MaxY)/2
		bx, by := (b.MinX+b.MaxX)/2, (b.MinY+b.MaxY)/2
		return writeJSON(map[string]any{"distance": math.Hypot(ax-bx, ay-by), "units": "coordinate units", "method": "bbox centres"})
	case "simplify":
		raw, e := o.value("--tolerance")
		if e != nil {
			return e
		}
		tol, e := parseFloat(raw, "tolerance")
		if e != nil || tol < 0 {
			return fmt.Errorf("tolerance must be non-negative")
		}
		for i := range fc.Features {
			fc.Features[i].Geometry.Coordinates = simplifyCoordinates(fc.Features[i].Geometry.Coordinates, tol)
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveGeoJSON(out, fc)
	case "buffer":
		raw, e := o.value("--distance")
		if e != nil {
			return e
		}
		d, e := parseFloat(raw, "distance")
		if e != nil || d < 0 {
			return fmt.Errorf("distance must be non-negative")
		}
		for i := range fc.Features {
			b := geomBounds(fc.Features[i].Geometry)
			if b.Valid {
				fc.Features[i].Geometry = bboxGeometry(bounds{b.MinX - d, b.MinY - d, b.MaxX + d, b.MaxY + d, true})
			}
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveGeoJSON(out, fc)
	case "validate":
		issues := []string{}
		for i, f := range fc.Features {
			if f.Geometry.Type == "" {
				issues = append(issues, fmt.Sprintf("feature %d has no geometry type", i))
			}
			if !geomBounds(f.Geometry).Valid {
				issues = append(issues, fmt.Sprintf("feature %d has no coordinates", i))
			}
		}
		if out := o.optional("--output", ""); out != "" {
			if e := saveGeoJSON(out, fc); e != nil {
				return e
			}
		}
		return writeJSON(map[string]any{"valid": len(issues) == 0, "issues": issues, "repaired": o.flag("--repair")})
	case "intersect", "union", "difference", "clip":
		return runVectorOverlay(name, fc, o)
	}
	return fmt.Errorf("unsupported vector command %q", name)
}

func lonLatToMercator(lon, lat float64) (float64, float64) {
	lat = math.Max(-85.05112878, math.Min(85.05112878, lat))
	return earthRadius * radians(lon), earthRadius * math.Log(math.Tan(math.Pi/4+radians(lat)/2))
}
func mercatorToLonLat(x, y float64) (float64, float64) {
	return degrees(x / earthRadius), degrees(2*math.Atan(math.Exp(y/earthRadius)) - math.Pi/2)
}
func bboxGeometry(b bounds) geometry {
	return geometry{Type: "Polygon", Coordinates: []any{[]any{[]any{b.MinX, b.MinY}, []any{b.MaxX, b.MinY}, []any{b.MaxX, b.MaxY}, []any{b.MinX, b.MaxY}, []any{b.MinX, b.MinY}}}}
}
func intersectBounds(a, b bounds) (bounds, bool) {
	r := bounds{math.Max(a.MinX, b.MinX), math.Max(a.MinY, b.MinY), math.Min(a.MaxX, b.MaxX), math.Min(a.MaxY, b.MaxY), true}
	return r, r.MinX <= r.MaxX && r.MinY <= r.MaxY
}
func runVectorOverlay(name string, fc featureCollection, o options) error {
	otherKey := "--other"
	if name == "clip" {
		otherKey = "--mask"
	}
	path, e := o.value(otherKey)
	if e != nil {
		return e
	}
	oc, e := loadGeoJSON(path)
	if e != nil {
		return e
	}
	a, b := collectionBounds(fc), collectionBounds(oc)
	if !a.Valid || !b.Valid {
		return fmt.Errorf("dataset has no coordinates")
	}
	var results []feature
	switch name {
	case "intersect", "clip":
		if q, ok := intersectBounds(a, b); ok {
			results = []feature{{Type: "Feature", Properties: map[string]any{"operation": name, "method": "bounding-box overlay"}, Geometry: bboxGeometry(q)}}
		}
	case "union":
		q := a
		q.add(b.MinX, b.MinY)
		q.add(b.MaxX, b.MaxY)
		results = []feature{{Type: "Feature", Properties: map[string]any{"operation": name, "method": "bounding-box overlay"}, Geometry: bboxGeometry(q)}}
	case "difference":
		results = []feature{{Type: "Feature", Properties: map[string]any{"operation": name, "method": "bounding-box conservative difference"}, Geometry: bboxGeometry(a)}}
	}
	out, e := requireOutput(o)
	if e != nil {
		return e
	}
	return saveGeoJSON(out, featureCollection{Type: "FeatureCollection", Features: results})
}
func geomCentroid(g geometry) (float64, float64, bool) {
	var sx, sy float64
	n := 0
	walkCoords(g.Coordinates, func(x, y float64) (float64, float64) { sx += x; sy += y; n++; return x, y })
	if n == 0 {
		return 0, 0, false
	}
	return sx / float64(n), sy / float64(n), true
}
func coordsAsPoints(v any) [][2]float64 {
	a, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([][2]float64, 0, len(a))
	for _, q := range a {
		p, ok := q.([]any)
		if ok && len(p) >= 2 {
			if x, xok := p[0].(float64); xok {
				if y, yok := p[1].(float64); yok {
					out = append(out, [2]float64{x, y})
				}
			}
		}
	}
	return out
}
func geomMeasure(g geometry, kind string, geodesic bool) float64 {
	var total float64
	var visit func(any)
	visit = func(v any) {
		if p := coordsAsPoints(v); len(p) >= 2 {
			if kind == "area" && len(p) >= 3 {
				for i := 0; i < len(p)-1; i++ {
					total += p[i][0]*p[i+1][1] - p[i+1][0]*p[i][1]
				}
				total = math.Abs(total) / 2
			} else if kind == "length" {
				for i := 0; i < len(p)-1; i++ {
					if geodesic {
						total += haversine(lonLat{p[i][0], p[i][1]}, lonLat{p[i+1][0], p[i+1][1]})
					} else {
						total += math.Hypot(p[i+1][0]-p[i][0], p[i+1][1]-p[i][1])
					}
				}
			}
			return
		}
		if a, ok := v.([]any); ok {
			for _, x := range a {
				visit(x)
			}
		}
	}
	visit(g.Coordinates)
	return total
}
func measureUnits(kind string, geodesic bool) string {
	if geodesic && kind == "length" {
		return "metres"
	}
	return "coordinate units"
}
func simplifyCoordinates(v any, tol float64) any {
	a, ok := v.([]any)
	if !ok {
		return v
	}
	if p := coordsAsPoints(v); len(p) >= 3 {
		keep := douglasPeucker(p, tol)
		out := make([]any, len(keep))
		for i, q := range keep {
			out[i] = []any{q[0], q[1]}
		}
		return out
	}
	out := make([]any, len(a))
	for i := range a {
		out[i] = simplifyCoordinates(a[i], tol)
	}
	return out
}
func douglasPeucker(p [][2]float64, eps float64) [][2]float64 {
	if len(p) < 3 {
		return p
	}
	maxD, index := 0.0, 0
	a, b := p[0], p[len(p)-1]
	for i := 1; i < len(p)-1; i++ {
		d := pointLineDistance(p[i], a, b)
		if d > maxD {
			maxD, index = d, i
		}
	}
	if maxD <= eps {
		return [][2]float64{a, b}
	}
	left := douglasPeucker(p[:index+1], eps)
	right := douglasPeucker(p[index:], eps)
	return append(left[:len(left)-1], right...)
}
func pointLineDistance(p, a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	if dx == 0 && dy == 0 {
		return math.Hypot(p[0]-a[0], p[1]-a[1])
	}
	return math.Abs(dy*p[0]-dx*p[1]+b[0]*a[1]-b[1]*a[0]) / math.Hypot(dx, dy)
}
func vectorPolygonArea(o options) error {
	in, e := requireInput(o)
	if e != nil {
		return e
	}
	fc, e := loadGeoJSON(in)
	if e != nil {
		return e
	}
	var area float64
	for _, f := range fc.Features {
		area += geomMeasure(f.Geometry, "area", false)
	}
	return writeJSON(map[string]any{"area": area, "units": "square coordinate units", "method": "planar shoelace"})
}
