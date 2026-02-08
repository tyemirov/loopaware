package main

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidServeMode = errors.New("invalid serve mode")

type ServeMode string

const (
	ServeModeMonolith ServeMode = "monolith"
	ServeModeWeb      ServeMode = "web"
	ServeModeAPI      ServeMode = "api"
)

func ParseServeMode(rawInput string) (ServeMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(rawInput))
	if normalized == "" {
		return ServeModeMonolith, nil
	}

	mode := ServeMode(normalized)
	switch mode {
	case ServeModeMonolith, ServeModeWeb, ServeModeAPI:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidServeMode, rawInput)
	}
}
