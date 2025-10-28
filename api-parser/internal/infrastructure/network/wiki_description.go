package network

import (
	"api-parser/internal/domain"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"

	"golang.org/x/sync/errgroup"
)

var threadPoolNumWorkers = runtime.NumCPU()

func getWikiPlaceInfo(ctx context.Context, client *http.Client, pageID int) (domain.PlaceInfo, error) {
	q := url.Values{}
	q.Set("action", "query")
	q.Set("prop", "extracts|info")
	q.Set("inprop", "url") // даст fullurl
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
	if err := doJSON(client, req, &resp); err != nil {
		return domain.PlaceInfo{}, err
	}
	if len(resp.Query.Pages) == 0 {
		return domain.PlaceInfo{}, fmt.Errorf("page %d missing", pageID)
	}
	return resp.Query.Pages[0], nil
}

func ReadPlaceDescriptionAsync(ctx context.Context, client *http.Client, wikiChan <-chan Result[domain.WikiGeosearchResp]) <-chan Result[[]domain.PlaceInfo] {
	out := make(chan Result[[]domain.PlaceInfo], 1)
	go func() {
		var res Result[domain.WikiGeosearchResp]
		select {
		case res = <-wikiChan:
		case <-ctx.Done():
		}

		if res.Err != nil {
			select {
			case out <- Result[[]domain.PlaceInfo]{Err: res.Err}:
			case <-ctx.Done():
			}
			return
		}

		geoSearch := res.Value.Query.Geosearch
		if len(geoSearch) == 0 {
			select {
			case out <- Result[[]domain.PlaceInfo]{Err: fmt.Errorf("no results found")}:
			case <-ctx.Done():
			}
			return
		}

		type job struct {
			id  int
			idx int
		}

		jobs := make(chan job)
		placeInfo := make([]domain.PlaceInfo, len(geoSearch))
		g, gctx := errgroup.WithContext(ctx)
		for range threadPoolNumWorkers {
			g.Go(func() error {
				for j := range jobs {
					select {
					case <-gctx.Done():
						return gctx.Err()
					default:
					}
					info, err := getWikiPlaceInfo(gctx, client, j.id)
					if err != nil {
						return fmt.Errorf("get place info: %w", err)
					}
					placeInfo[j.idx] = info
				}
				return nil
			})
		}

		g.Go(func() error {
			defer close(jobs)
			for i, gs := range geoSearch {
				select {
				case jobs <- job{id: gs.PageID, idx: i}:
				case <-gctx.Done():
					return gctx.Err()
				}
			}
			return nil
		})
		if err := g.Wait(); err != nil {
			select {
			case out <- Result[[]domain.PlaceInfo]{Err: err}:
			case <-gctx.Done():
			}
			return
		}
		select {
		case out <- Result[[]domain.PlaceInfo]{Value: placeInfo, Err: nil}:
		case <-ctx.Done():
		}
	}()
	return out
}
