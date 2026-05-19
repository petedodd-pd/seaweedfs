package filer

import (
	"errors"
	"strings"
	"time"
)

type AtimeMode string

const (
	AtimeModeOff      AtimeMode = "off"
	AtimeModeRelatime AtimeMode = "relatime"
	AtimeModeStrict   AtimeMode = "strict"

	DefaultRelatimeThreshold = 24 * time.Hour
)

var ErrInvalidAtimeMode = errors.New("invalid atime mode: expected one of off, relatime, strict")

type AtimePolicy struct {
	Mode               AtimeMode
	RelatimeThreshold  time.Duration
}

func NewAtimePolicy(mode string, threshold time.Duration) (*AtimePolicy, error) {
	parsed, err := ParseAtimeMode(mode)
	if err != nil {
		return nil, err
	}
	if threshold <= 0 {
		threshold = DefaultRelatimeThreshold
	}
	return &AtimePolicy{Mode: parsed, RelatimeThreshold: threshold}, nil
}

func ParseAtimeMode(s string) (AtimeMode, error) {
	switch AtimeMode(strings.ToLower(strings.TrimSpace(s))) {
	case AtimeModeOff:
		return AtimeModeOff, nil
	case AtimeModeRelatime, "":
		return AtimeModeRelatime, nil
	case AtimeModeStrict:
		return AtimeModeStrict, nil
	}
	return "", ErrInvalidAtimeMode
}

func (p *AtimePolicy) ShouldUpdate(existing Attr, candidate time.Time) bool {
	if p == nil || p.Mode == AtimeModeOff {
		return false
	}
	if candidate.IsZero() {
		return false
	}
	if existing.Atime.IsZero() {
		return true
	}
	if !candidate.After(existing.Atime) {
		return false
	}
	if p.Mode == AtimeModeStrict {
		return true
	}
	if !existing.Mtime.IsZero() && existing.Atime.Before(existing.Mtime) {
		return true
	}
	if !existing.Ctime.IsZero() && existing.Atime.Before(existing.Ctime) {
		return true
	}
	return candidate.Sub(existing.Atime) >= p.RelatimeThreshold
}
