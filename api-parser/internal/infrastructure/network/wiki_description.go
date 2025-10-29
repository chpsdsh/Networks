package network

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/network/utils"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/sync/errgroup"
)

const threadPoolNumWorkers = 10

func getWikiPlaceInfo(ctx context.Context, client *http.Client, pageID int) (domain.PlaceInfo, error) {
	q := url.Values{}
	q.Set("action", "query")
	q.Set("prop", "extracts|info")
	q.Set("inprop", "url")
	q.Set("exintro", "1")
	q.Set("explaintext", "1")
	q.Set("pageids", strconv.Itoa(pageID))
	q.Set("format", "json")
	q.Set("formatversion", "2")

	u := "https://en.wikipedia.org/w/api.php?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return domain.PlaceInfo{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "api-parser/1.0")

	var resp domain.WikiPlaceInfo
	if err := utils.DoJSON(client, req, &resp); err != nil {
		return domain.PlaceInfo{}, err
	}

	if len(resp.Query.Pages) == 0 {
		return domain.PlaceInfo{}, fmt.Errorf("page %d missing", pageID)
	}
	return resp.Query.Pages[0], nil
}

func ReadPlaceDescriptionAsync(ctx context.Context, client *http.Client, wikiChan <-chan utils.Result[domain.WikiGeoSearchResp]) <-chan utils.Result[[]domain.WikiGeoSearchAndPlaceInfo] {
	out := make(chan utils.Result[[]domain.WikiGeoSearchAndPlaceInfo], 1)
	go func() {
		defer close(out)
		var wikiGeoSearchResult utils.Result[domain.WikiGeoSearchResp]
		select {
		case wikiGeoSearchResult = <-wikiChan:
		case <-ctx.Done():
		}

		if wikiGeoSearchResult.Err != nil {
			select {
			case out <- utils.Result[[]domain.WikiGeoSearchAndPlaceInfo]{Err: wikiGeoSearchResult.Err}:
			case <-ctx.Done():
			}
			return
		}

		geoSearch := wikiGeoSearchResult.Value.Query.GeoSearch
		if len(geoSearch) == 0 {
			select {
			case out <- utils.Result[[]domain.WikiGeoSearchAndPlaceInfo]{Err: fmt.Errorf("no results found")}:
			case <-ctx.Done():
			}
			return
		}

		getDescriptionAsync(geoSearch, ctx, client, out)
	}()
	return out
}

func getDescriptionAsync(geoSearch []domain.GeoSearch, ctx context.Context, client *http.Client, out chan utils.Result[[]domain.WikiGeoSearchAndPlaceInfo]) {
	type job struct {
		geoSearch domain.GeoSearch
		idx       int
	}

	jobs := make(chan job)
	geoSearchAndPlaceInfo := make([]domain.WikiGeoSearchAndPlaceInfo, len(geoSearch))
	g, gCtx := errgroup.WithContext(ctx)
	for range threadPoolNumWorkers {
		g.Go(func() error {
			for j := range jobs {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				default:
				}
				info, err := getWikiPlaceInfo(gCtx, client, j.geoSearch.PageID)
				if err != nil {
					return fmt.Errorf("get place info: %w", err)
				}
				geoSearchAndPlaceInfo[j.idx] = domain.WikiGeoSearchAndPlaceInfo{GeoSearch: j.geoSearch, PlaceInfo: info}
			}
			return nil
		})
	}

	g.Go(func() error {
		defer close(jobs)
		for i, gs := range geoSearch {
			select {
			case jobs <- job{geoSearch: gs, idx: i}:
			case <-gCtx.Done():
				return gCtx.Err()
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		select {
		case out <- utils.Result[[]domain.WikiGeoSearchAndPlaceInfo]{Err: err}:
		case <-ctx.Done():
		}
		return
	}

	select {
	case out <- utils.Result[[]domain.WikiGeoSearchAndPlaceInfo]{Value: geoSearchAndPlaceInfo}:
	case <-ctx.Done():
		return
	}
}
