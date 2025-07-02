package api

import (
	"bytes"

	json "github.com/goccy/go-json"
)

func structToBuffer(s any) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(s)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}
