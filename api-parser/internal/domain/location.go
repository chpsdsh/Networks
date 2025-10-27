package domain

type Location struct {
	Name string  `json:"name"`
	Id   int     `json:"osm_id"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}
