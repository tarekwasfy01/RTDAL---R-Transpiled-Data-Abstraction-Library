package operations

import (
	"archive/zip"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifiers(t *testing.T) {
	values := []float64{1, 2, 3, 4, 10, 11, 12}
	for _, name := range []string{"equal", "quantile", "pretty", "jenks"} {
		breaks := make([]float64, 4)
		switch name {
		case "jenks":
			breaks = jenksBreaks(values, 3)
		case "equal":
			for i := range breaks {
				breaks[i] = values[0] + (values[len(values)-1]-values[0])*float64(i)/3
			}
		default:
			breaks[0], breaks[3] = values[0], values[len(values)-1]
		}
		if len(breaks) != 4 {
			t.Fatalf("%s returned %d breaks", name, len(breaks))
		}
	}
}

func TestGeodesy(t *testing.T) {
	berlin := lonLat{13.405, 52.52}
	paris := lonLat{2.3522, 48.8566}
	d := haversine(berlin, paris)
	if d < 870000 || d > 890000 {
		t.Fatalf("Berlin-Paris distance = %f", d)
	}
	dst := destination(berlin, initialBearing(berlin, paris), d)
	if math.Abs(dst.Lat-paris.Lat) > .01 || math.Abs(dst.Lon-paris.Lon) > .01 {
		t.Fatalf("destination = %#v", dst)
	}
}

func TestGeoJSONAndRasterRoundTrips(t *testing.T) {
	dir := t.TempDir()
	fc := featureCollection{Type: "FeatureCollection", Features: []feature{{Type: "Feature", Geometry: geometry{Type: "Point", Coordinates: []any{13.4, 52.5}}}}}
	vectorPath := filepath.Join(dir, "test.geojson")
	if err := saveGeoJSON(vectorPath, fc); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGeoJSON(vectorPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Features) != 1 || !collectionBounds(loaded).Valid {
		t.Fatal("GeoJSON did not round-trip")
	}
	grid := rasterGrid{Width: 2, Height: 2, Bands: 1, XMin: 0, YMin: 0, CellX: 1, CellY: 1, Data: [][]float64{{1, 2, 3, 4}}}
	for _, ext := range []string{".json", ".asc"} {
		path := filepath.Join(dir, "grid"+ext)
		if err := saveRaster(path, grid); err != nil {
			t.Fatal(err)
		}
		got, err := loadRaster(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Width != 2 || got.Height != 2 || len(got.Data[0]) != 4 {
			t.Fatalf("bad %s round-trip", ext)
		}
	}
}

func TestGTFSAndCubeRoundTrips(t *testing.T) {
	dir := t.TempDir()
	feed := filepath.Join(dir, "feed.zip")
	tables := []gtfsTable{{Name: "agency.txt", Header: []string{"agency_id", "agency_name"}, Rows: [][]string{{"1", "Demo"}}}, {Name: "stops.txt", Header: []string{"stop_id", "stop_name"}, Rows: [][]string{{"s", "Stop"}}}, {Name: "routes.txt", Header: []string{"route_id"}, Rows: [][]string{{"r"}}}, {Name: "trips.txt", Header: []string{"route_id", "trip_id"}, Rows: [][]string{{"r", "t"}}}, {Name: "stop_times.txt", Header: []string{"trip_id", "stop_id"}, Rows: [][]string{{"t", "s"}}}}
	if err := writeGTFSZip(feed, tables); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGTFS(feed)
	if err != nil {
		t.Fatal(err)
	}
	if issues := validateGTFS(loaded); len(issues) != 0 {
		t.Fatal(issues)
	}
	z, err := zip.OpenReader(feed)
	if err != nil {
		t.Fatal(err)
	}
	z.Close()
	c := cube{Dimensions: []string{"x", "y"}, Shape: []int{2, 2}, Data: []float64{1, 2, 3, 4}}
	path := filepath.Join(dir, "cube.json")
	if err := saveCube(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := loadCube(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 4 {
		t.Fatal("cube did not round-trip")
	}
	if _, err = os.Stat(feed); err != nil {
		t.Fatal(err)
	}
}
