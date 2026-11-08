package geo

import "math"

const (
	EarthRadiusKM         = 6371.0088
	DefaultDispatchRadius = 5.0
	MaxDispatchRadiusKM   = 5.0
)

type Point struct {
	Latitude  float64
	Longitude float64
}

func Valid(point Point) bool {
	if math.IsNaN(point.Latitude) || math.IsNaN(point.Longitude) ||
		math.IsInf(point.Latitude, 0) || math.IsInf(point.Longitude, 0) {
		return false
	}
	if point.Latitude < -90 || point.Latitude > 90 || point.Longitude < -180 || point.Longitude > 180 {
		return false
	}
	return !(point.Latitude == 0 && point.Longitude == 0)
}

func DistanceKM(from, to Point) float64 {
	lat1 := from.Latitude * math.Pi / 180
	lat2 := to.Latitude * math.Pi / 180
	deltaLat := (to.Latitude - from.Latitude) * math.Pi / 180
	deltaLng := (to.Longitude - from.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	a = math.Min(1, math.Max(0, a))
	return EarthRadiusKM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func RoundKM(value float64) float64 {
	return math.Round(value*10) / 10
}
