package domain

type GeoResponse struct {
	Hits []struct {
		OSMId int    `json:"osm_id"`
		Name  string `json:"name"`
		Point Point  `json:"point"`
	} `json:"hits"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
