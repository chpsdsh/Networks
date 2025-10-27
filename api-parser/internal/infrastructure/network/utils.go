package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Result[T any] struct {
	Value T
	Err   error
}

func doJSON(client *http.Client, request *http.Request, out any) error {
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("error decoding: %w", err)
	}
	return nil
}
