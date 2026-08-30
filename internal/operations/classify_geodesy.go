package operations

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func numericList(raw string) ([]float64, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' })
	if len(parts) == 0 {
		return nil, fmt.Errorf("input contains no numeric values")
	}
	values := make([]float64, len(parts))
	for i, p := range parts {
		v, e := strconv.ParseFloat(p, 64)
		if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid numeric value %q", p)
		}
		values[i] = v
	}
	return values, nil
}
func classesOption(o options) (int, error) {
	raw, e := o.value("--classes")
	if e != nil {
		return 0, e
	}
	n, e := strconv.Atoi(raw)
	if e != nil || n < 1 {
		return 0, fmt.Errorf("--classes must be a positive integer")
	}
	return n, nil
}
func runClassify(name string, o options) error {
	raw, e := o.value("--input")
	if e != nil {
		return e
	}
	values, e := numericList(raw)
	if e != nil {
		return e
	}
	n, e := classesOption(o)
	if e != nil {
		return e
	}
	sort.Float64s(values)
	breaks := make([]float64, n+1)
	switch name {
	case "equal":
		lo, hi := values[0], values[len(values)-1]
		for i := range breaks {
			breaks[i] = lo + (hi-lo)*float64(i)/float64(n)
		}
	case "quantile":
		for i := range breaks {
			p := float64(i) * float64(len(values)-1) / float64(n)
			lo, hi := int(math.Floor(p)), int(math.Ceil(p))
			f := p - float64(lo)
			breaks[i] = values[lo]*(1-f) + values[hi]*f
		}
	case "pretty":
		lo, hi := values[0], values[len(values)-1]
		step := niceNumber((hi - lo) / float64(n))
		if step == 0 {
			step = 1
		}
		start := math.Floor(lo/step) * step
		for i := range breaks {
			breaks[i] = start + float64(i)*step
		}
	case "jenks":
		breaks = jenksBreaks(values, n)
	default:
		return fmt.Errorf("unsupported classify command %q", name)
	}
	return writeJSON(map[string]any{"method": name, "classes": n, "breaks": breaks})
}
func niceNumber(v float64) float64 {
	if v <= 0 {
		return 0
	}
	p := math.Pow(10, math.Floor(math.Log10(v)))
	f := v / p
	switch {
	case f <= 1:
		f = 1
	case f <= 2:
		f = 2
	case f <= 5:
		f = 5
	default:
		f = 10
	}
	return f * p
}
func jenksBreaks(data []float64, classes int) []float64 {
	if classes >= len(data) {
		out := append([]float64(nil), data...)
		for len(out) < classes+1 {
			out = append(out, data[len(data)-1])
		}
		return out[:classes+1]
	}
	n := len(data)
	lower := make([][]int, n+1)
	variance := make([][]float64, n+1)
	for i := range lower {
		lower[i] = make([]int, classes+1)
		variance[i] = make([]float64, classes+1)
		for j := 1; j <= classes; j++ {
			variance[i][j] = math.Inf(1)
		}
	}
	for j := 1; j <= classes; j++ {
		lower[1][j] = 1
		variance[1][j] = 0
	}
	for l := 2; l <= n; l++ {
		var sum, sumSq float64
		w := 0
		for m := 1; m <= l; m++ {
			i3 := l - m + 1
			v := data[i3-1]
			w++
			sum += v
			sumSq += v * v
			vari := sumSq - sum*sum/float64(w)
			if i3 > 1 {
				for j := 2; j <= classes; j++ {
					cand := vari + variance[i3-1][j-1]
					if cand < variance[l][j] {
						lower[l][j] = i3
						variance[l][j] = cand
					}
				}
			}
		}
		lower[l][1] = 1
		variance[l][1] = sumSq - sum*sum/float64(w)
	}
	out := make([]float64, classes+1)
	out[0] = data[0]
	out[classes] = data[n-1]
	k := n
	for j := classes; j >= 2; j-- {
		idx := lower[k][j] - 2
		if idx < 0 {
			idx = 0
		}
		out[j-1] = data[idx]
		k = lower[k][j] - 1
	}
	return out
}

type lonLat struct{ Lon, Lat float64 }

func parseLonLat(raw string) (lonLat, error) {
	v, e := numericList(raw)
	if e != nil || len(v) != 2 {
		return lonLat{}, fmt.Errorf("coordinate must be lon,lat")
	}
	if v[1] < -90 || v[1] > 90 {
		return lonLat{}, fmt.Errorf("latitude out of range")
	}
	return lonLat{v[0], v[1]}, nil
}

const earthRadius = 6371008.8

func runGeodesy(name string, o options) error {
	if name == "polygon-area" {
		return vectorPolygonArea(o)
	}
	fr, e := o.value("--from")
	if e != nil {
		return e
	}
	from, e := parseLonLat(fr)
	if e != nil {
		return e
	}
	switch name {
	case "distance", "bearing":
		tr, e := o.value("--to")
		if e != nil {
			return e
		}
		to, e := parseLonLat(tr)
		if e != nil {
			return e
		}
		if name == "distance" {
			m := haversine(from, to)
			return writeJSON(map[string]any{"metres": m, "kilometres": m / 1000, "method": "haversine"})
		}
		return writeJSON(map[string]any{"degrees": initialBearing(from, to)})
	case "destination":
		br, e := o.value("--bearing")
		if e != nil {
			return e
		}
		dr, e := o.value("--distance")
		if e != nil {
			return e
		}
		b, e := parseFloat(br, "bearing")
		if e != nil {
			return e
		}
		d, e := parseFloat(dr, "distance")
		if e != nil || d < 0 {
			return fmt.Errorf("distance must be non-negative")
		}
		dst := destination(from, b, d)
		return writeJSON(map[string]any{"longitude": dst.Lon, "latitude": dst.Lat})
	}
	return fmt.Errorf("unsupported geodesy command %q", name)
}
func radians(v float64) float64 { return v * math.Pi / 180 }
func degrees(v float64) float64 { return v * 180 / math.Pi }
func haversine(a, b lonLat) float64 {
	p1, p2 := radians(a.Lat), radians(b.Lat)
	dp := p2 - p1
	dl := radians(b.Lon - a.Lon)
	h := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
func initialBearing(a, b lonLat) float64 {
	p1, p2 := radians(a.Lat), radians(b.Lat)
	dl := radians(b.Lon - a.Lon)
	y := math.Sin(dl) * math.Cos(p2)
	x := math.Cos(p1)*math.Sin(p2) - math.Sin(p1)*math.Cos(p2)*math.Cos(dl)
	return math.Mod(degrees(math.Atan2(y, x))+360, 360)
}
func destination(a lonLat, bearing, metres float64) lonLat {
	p, l, b, d := radians(a.Lat), radians(a.Lon), radians(bearing), metres/earthRadius
	p2 := math.Asin(math.Sin(p)*math.Cos(d) + math.Cos(p)*math.Sin(d)*math.Cos(b))
	l2 := l + math.Atan2(math.Sin(b)*math.Sin(d)*math.Cos(p), math.Cos(d)-math.Sin(p)*math.Sin(p2))
	return lonLat{math.Mod(degrees(l2)+540, 360) - 180, degrees(p2)}
}
