package network

import (
	"api-parser/internal/domain"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func ReadGeoDataAsync(ctx context.Context, client *http.Client, query string) <-chan Result[[]domain.Location] {
	out := make(chan Result[[]domain.Location], 1)
	go func() {
		defer close(out)

		key := os.Getenv("GRAPHOPPER_KEY")
		if key == "" {
			select {
			case out <- Result[[]domain.Location]{Err: fmt.Errorf("env var GRAPHOPPER_KEY not set")}:
			case <-ctx.Done():
			}
			return
		}

		u := fmt.Sprintf(
			"https://graphhopper.com/api/1/geocode?q=%s&key=%s",
			url.QueryEscape(query), key,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			select {
			case out <- Result[[]domain.Location]{Err: fmt.Errorf("error creating request: %w", err)}:
			case <-ctx.Done():
			}
			return
		}

		var geoResponse domain.GeoResponse
		if err := doJSON(client, req, &geoResponse); err != nil {
			out <- Result[[]domain.Location]{Err: fmt.Errorf("error doing request: %w", err)}
			return
		}

		locs := make([]domain.Location, 0, len(geoResponse.Hits))

		for _, h := range geoResponse.Hits {
			locs = append(locs, domain.Location{
				Name: h.Name,
				Id:   h.OSMId,
				Lat:  h.Point.Lat,
				Lng:  h.Point.Lng,
			})
		}

		select {
		case out <- Result[[]domain.Location]{Value: locs}:
		case <-ctx.Done():
			return
		}
	}()
	return out
}
