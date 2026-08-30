package operations

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type rasterGrid struct {
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Bands  int         `json:"bands"`
	XMin   float64     `json:"xmin"`
	YMin   float64     `json:"ymin"`
	CellX  float64     `json:"cell_x"`
	CellY  float64     `json:"cell_y"`
	CRS    string      `json:"crs,omitempty"`
	NoData *float64    `json:"nodata,omitempty"`
	Data   [][]float64 `json:"data"`
}

func loadRaster(path string) (rasterGrid, error) {
	data, e := os.ReadFile(path)
	if e != nil {
		return rasterGrid{}, e
	}
	if strings.EqualFold(filepath.Ext(path), ".json") {
		var r rasterGrid
		e = json.Unmarshal(data, &r)
		if e != nil {
			return r, e
		}
		if e = validateRaster(r); e != nil {
			return r, e
		}
		return r, nil
	}
	return parseASCIIGrid(string(data))
}
func validateRaster(r rasterGrid) error {
	if r.Width <= 0 || r.Height <= 0 {
		return fmt.Errorf("invalid raster dimensions")
	}
	if r.Bands == 0 {
		r.Bands = len(r.Data)
	}
	if len(r.Data) == 0 {
		return fmt.Errorf("raster has no bands")
	}
	for i, b := range r.Data {
		if len(b) != r.Width*r.Height {
			return fmt.Errorf("band %d has %d cells, expected %d", i+1, len(b), r.Width*r.Height)
		}
	}
	return nil
}
func parseASCIIGrid(text string) (rasterGrid, error) {
	s := bufio.NewScanner(strings.NewReader(text))
	headers := map[string]float64{}
	var values []float64
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) == 0 {
			continue
		}
		key := strings.ToLower(fields[0])
		if len(headers) < 6 && len(fields) == 2 && strings.Contains("ncols nrows xllcorner xllcenter yllcorner yllcenter cellsize nodata_value", key) {
			v, e := strconv.ParseFloat(fields[1], 64)
			if e != nil {
				return rasterGrid{}, e
			}
			headers[key] = v
			continue
		}
		for _, f := range fields {
			v, e := strconv.ParseFloat(f, 64)
			if e != nil {
				return rasterGrid{}, fmt.Errorf("invalid ASCII grid value %q", f)
			}
			values = append(values, v)
		}
	}
	w, h := int(headers["ncols"]), int(headers["nrows"])
	if w <= 0 || h <= 0 || len(values) != w*h {
		return rasterGrid{}, fmt.Errorf("invalid ASCII grid: dimensions %dx%d, %d values", w, h, len(values))
	}
	cell := headers["cellsize"]
	nd := headers["nodata_value"]
	return rasterGrid{Width: w, Height: h, Bands: 1, XMin: firstNonzero(headers["xllcorner"], headers["xllcenter"]), YMin: firstNonzero(headers["yllcorner"], headers["yllcenter"]), CellX: cell, CellY: cell, NoData: &nd, Data: [][]float64{values}}, nil
}
func firstNonzero(a, b float64) float64 {
	if a != 0 {
		return a
	}
	return b
}
func saveRaster(path string, r rasterGrid) error {
	if e := validateRaster(r); e != nil {
		return e
	}
	if strings.EqualFold(filepath.Ext(path), ".asc") {
		var b strings.Builder
		fmt.Fprintf(&b, "ncols %d\nnrows %d\nxllcorner %.12g\nyllcorner %.12g\ncellsize %.12g\n", r.Width, r.Height, r.XMin, r.YMin, r.CellX)
		nd := -9999.0
		if r.NoData != nil {
			nd = *r.NoData
		}
		fmt.Fprintf(&b, "NODATA_value %.12g\n", nd)
		for y := 0; y < r.Height; y++ {
			for x := 0; x < r.Width; x++ {
				if x > 0 {
					b.WriteByte(' ')
				}
				fmt.Fprintf(&b, "%.12g", r.Data[0][y*r.Width+x])
			}
			b.WriteByte('\n')
		}
		return os.WriteFile(path, []byte(b.String()), 0644)
	}
	data, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
func rasterInputs(o options) ([]string, error) {
	v := o.values("--input")
	if len(v) == 0 {
		return nil, fmt.Errorf("missing required option --input")
	}
	return v, nil
}
func runRaster(name string, o options) error {
	inputs, e := rasterInputs(o)
	if e != nil {
		return e
	}
	rasters := make([]rasterGrid, len(inputs))
	for i, p := range inputs {
		rasters[i], e = loadRaster(p)
		if e != nil {
			return fmt.Errorf("read raster %q: %w", p, e)
		}
	}
	r := rasters[0]
	switch name {
	case "info":
		stats := rasterStats(r, 0)
		return writeJSON(map[string]any{"format": rasterFormat(inputs[0]), "width": r.Width, "height": r.Height, "bands": len(r.Data), "extent": []float64{r.XMin, r.YMin, r.XMin + float64(r.Width)*r.CellX, r.YMin + float64(r.Height)*r.CellY}, "resolution": []float64{r.CellX, r.CellY}, "crs": r.CRS, "statistics": stats})
	case "statistics":
		band := 1
		if raw := o.optional("--band", ""); raw != "" {
			band, e = strconv.Atoi(raw)
			if e != nil || band < 1 || band > len(r.Data) {
				return fmt.Errorf("invalid band")
			}
		}
		return writeJSON(rasterStats(r, band-1))
	case "convert":
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "crop":
		raw, e := o.value("--bbox")
		if e != nil {
			return e
		}
		bb, e := numericList(raw)
		if e != nil || len(bb) != 4 {
			return fmt.Errorf("--bbox must be xmin,ymin,xmax,ymax")
		}
		r, e = cropRaster(r, bb)
		if e != nil {
			return e
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "resample":
		raw, e := o.value("--resolution")
		if e != nil {
			return e
		}
		res, e := numericList(raw)
		if e != nil || len(res) != 2 || res[0] <= 0 || res[1] <= 0 {
			return fmt.Errorf("--resolution must be positive x,y")
		}
		r = resampleRaster(r, res[0], res[1])
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "reproject":
		target, e := o.value("--target-crs")
		if e != nil {
			return e
		}
		if r.CRS == "" {
			return fmt.Errorf("input raster has no CRS")
		}
		r.CRS = target
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "mosaic":
		r, e = mosaicRasters(rasters)
		if e != nil {
			return e
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "calc":
		expr, e := o.value("--expression")
		if e != nil {
			return e
		}
		r, e = calculateRasters(rasters, expr)
		if e != nil {
			return e
		}
		out, e := requireOutput(o)
		if e != nil {
			return e
		}
		return saveRaster(out, r)
	case "tile":
		dir, e := requireOutput(o)
		if e != nil {
			return e
		}
		zoom, e := o.value("--zoom")
		if e != nil {
			return e
		}
		return tileRaster(r, dir, zoom)
	}
	return fmt.Errorf("unsupported raster command %q", name)
}
func rasterFormat(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return "RTDAL JSON raster"
	}
	return "ESRI ASCII Grid"
}
func rasterStats(r rasterGrid, band int) map[string]any {
	v := r.Data[band]
	min, max, sum, count := math.Inf(1), math.Inf(-1), 0.0, 0
	for _, x := range v {
		if r.NoData != nil && x == *r.NoData {
			continue
		}
		min = math.Min(min, x)
		max = math.Max(max, x)
		sum += x
		count++
	}
	mean := math.NaN()
	if count > 0 {
		mean = sum / float64(count)
	}
	return map[string]any{"band": band + 1, "count": count, "min": min, "max": max, "mean": mean}
}
func cropRaster(r rasterGrid, bb []float64) (rasterGrid, error) {
	x0 := maxInt(0, int(math.Floor((bb[0]-r.XMin)/r.CellX)))
	x1 := minInt(r.Width, int(math.Ceil((bb[2]-r.XMin)/r.CellX)))
	y0 := maxInt(0, int(math.Floor((bb[1]-r.YMin)/r.CellY)))
	y1 := minInt(r.Height, int(math.Ceil((bb[3]-r.YMin)/r.CellY)))
	if x0 >= x1 || y0 >= y1 {
		return r, fmt.Errorf("crop does not intersect raster")
	}
	out := r
	out.Width, out.Height = x1-x0, y1-y0
	out.XMin += float64(x0) * r.CellX
	out.YMin += float64(y0) * r.CellY
	out.Data = make([][]float64, len(r.Data))
	for b := range r.Data {
		out.Data[b] = make([]float64, out.Width*out.Height)
		for y := 0; y < out.Height; y++ {
			copy(out.Data[b][y*out.Width:(y+1)*out.Width], r.Data[b][(y+y0)*r.Width+x0:(y+y0)*r.Width+x1])
		}
	}
	return out, nil
}
func resampleRaster(r rasterGrid, cx, cy float64) rasterGrid {
	w := maxInt(1, int(math.Round(float64(r.Width)*r.CellX/cx)))
	h := maxInt(1, int(math.Round(float64(r.Height)*r.CellY/cy)))
	out := r
	out.Width, out.Height, out.CellX, out.CellY = w, h, cx, cy
	out.Data = make([][]float64, len(r.Data))
	for b := range r.Data {
		out.Data[b] = make([]float64, w*h)
		for y := 0; y < h; y++ {
			sy := minInt(r.Height-1, int(float64(y)*cy/r.CellY))
			for x := 0; x < w; x++ {
				sx := minInt(r.Width-1, int(float64(x)*cx/r.CellX))
				out.Data[b][y*w+x] = r.Data[b][sy*r.Width+sx]
			}
		}
	}
	return out
}
func mosaicRasters(rs []rasterGrid) (rasterGrid, error) {
	if len(rs) == 1 {
		return rs[0], nil
	}
	base := rs[0]
	for _, r := range rs[1:] {
		if r.Height != base.Height || r.CellX != base.CellX || r.CellY != base.CellY || len(r.Data) != len(base.Data) {
			return base, fmt.Errorf("mosaic inputs must have equal height, resolution and band count")
		}
		for b := range base.Data {
			rows := make([]float64, 0, (base.Width+r.Width)*base.Height)
			for y := 0; y < base.Height; y++ {
				rows = append(rows, base.Data[b][y*base.Width:(y+1)*base.Width]...)
				rows = append(rows, r.Data[b][y*r.Width:(y+1)*r.Width]...)
			}
			base.Data[b] = rows
		}
		base.Width += r.Width
	}
	return base, nil
}
func calculateRasters(rs []rasterGrid, expr string) (rasterGrid, error) {
	base := rs[0]
	for _, r := range rs {
		if r.Width != base.Width || r.Height != base.Height {
			return base, fmt.Errorf("calc inputs must have equal dimensions")
		}
	}
	op := ""
	for _, candidate := range []string{"+", "-", "*", "/"} {
		if strings.Contains(expr, candidate) {
			op = candidate
			break
		}
	}
	if op == "" {
		return base, fmt.Errorf("expression supports A+B, A-B, A*B or A/B")
	}
	parts := strings.SplitN(strings.ReplaceAll(expr, " ", ""), op, 2)
	if len(parts) != 2 {
		return base, fmt.Errorf("invalid expression")
	}
	left, e := rasterOperand(parts[0], rs)
	if e != nil {
		return base, e
	}
	right, e := rasterOperand(parts[1], rs)
	if e != nil {
		return base, e
	}
	out := base
	out.Data = [][]float64{make([]float64, base.Width*base.Height)}
	for i := range out.Data[0] {
		a, b := left(i), right(i)
		switch op {
		case "+":
			out.Data[0][i] = a + b
		case "-":
			out.Data[0][i] = a - b
		case "*":
			out.Data[0][i] = a * b
		case "/":
			if b == 0 {
				return base, fmt.Errorf("division by zero at cell %d", i)
			}
			out.Data[0][i] = a / b
		}
	}
	out.Bands = 1
	return out, nil
}
func rasterOperand(raw string, rs []rasterGrid) (func(int) float64, error) {
	if len(raw) == 1 && raw[0] >= 'A' && raw[0] <= 'Z' {
		idx := int(raw[0] - 'A')
		if idx >= len(rs) {
			return nil, fmt.Errorf("expression references missing input %s", raw)
		}
		return func(i int) float64 { return rs[idx].Data[0][i] }, nil
	}
	v, e := strconv.ParseFloat(raw, 64)
	if e != nil {
		return nil, fmt.Errorf("invalid expression operand %q", raw)
	}
	return func(int) float64 { return v }, nil
}
func tileRaster(r rasterGrid, dir, zoom string) error {
	parts := strings.Split(zoom, ":")
	if len(parts) != 2 {
		return fmt.Errorf("--zoom must be min:max")
	}
	minZ, e := strconv.Atoi(parts[0])
	if e != nil {
		return e
	}
	maxZ, e := strconv.Atoi(parts[1])
	if e != nil || maxZ < minZ {
		return fmt.Errorf("invalid zoom range")
	}
	for z := minZ; z <= maxZ; z++ {
		folder := filepath.Join(dir, strconv.Itoa(z), "0")
		if e := os.MkdirAll(folder, 0755); e != nil {
			return e
		}
		if e := saveRaster(filepath.Join(folder, "0.json"), r); e != nil {
			return e
		}
	}
	return nil
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
