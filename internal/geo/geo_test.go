package geo

import "testing"

func TestDistanceKM(t *testing.T) {
	beijing := Point{Latitude: 39.9042, Longitude: 116.4074}
	oneDegreeNorth := Point{Latitude: 40.9042, Longitude: 116.4074}
	if got := DistanceKM(beijing, oneDegreeNorth); got < 111 || got > 112 {
		t.Fatalf("DistanceKM() = %.3f, want about 111.2", got)
	}
	if got := DistanceKM(beijing, beijing); got != 0 {
		t.Fatalf("DistanceKM(same) = %.6f, want 0", got)
	}
}

func TestValidRejectsMissingCoordinates(t *testing.T) {
	if Valid(Point{}) {
		t.Fatal("zero coordinates should be treated as missing")
	}
	if Valid(Point{Latitude: 91, Longitude: 116}) {
		t.Fatal("out-of-range latitude should be rejected")
	}
	if !Valid(Point{Latitude: 39.9042, Longitude: 116.4074}) {
		t.Fatal("valid Beijing coordinate was rejected")
	}
}
