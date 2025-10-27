package domain

type GeoResponse struct {
	Hits []hit `json:"hits"`
}

type hit struct {
	OSMId int    `json:"osm_id"`
	Name  string `json:"name"`
	Point struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"point"`
}
