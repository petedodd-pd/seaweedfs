package filer

import (
	"testing"
	"time"
)

func TestAtimePolicy_Off_NeverUpdates(t *testing.T) {
	policy := &AtimePolicy{Mode: AtimeModeOff}
	now := time.Now()
	existing := Attr{Atime: now.Add(-48 * time.Hour), Mtime: now.Add(-48 * time.Hour), Ctime: now.Add(-48 * time.Hour)}
	if policy.ShouldUpdate(existing, now) {
		t.Fatal("off mode must not update")
	}
}

func TestAtimePolicy_Strict_UpdatesWhenCandidateAdvances(t *testing.T) {
	policy := &AtimePolicy{Mode: AtimeModeStrict}
	existing := Attr{
		Atime: time.Unix(1_700_000_000, 0),
		Mtime: time.Unix(1_700_000_000, 0),
		Ctime: time.Unix(1_700_000_000, 0),
	}
	if !policy.ShouldUpdate(existing, time.Unix(1_700_000_001, 0)) {
		t.Fatal("strict must update when candidate is newer")
	}
	if policy.ShouldUpdate(existing, existing.Atime) {
		t.Fatal("strict must not update when candidate equals existing atime")
	}
}

func TestAtimePolicy_Relatime_AtimeOlderThanMtime(t *testing.T) {
	policy := &AtimePolicy{Mode: AtimeModeRelatime, RelatimeThreshold: 24 * time.Hour}
	existing := Attr{
		Atime: time.Unix(1_700_000_000, 0),
		Mtime: time.Unix(1_700_500_000, 0),
		Ctime: time.Unix(1_700_500_000, 0),
	}
	if !policy.ShouldUpdate(existing, time.Unix(1_700_500_001, 0)) {
		t.Fatal("relatime must update when atime < mtime")
	}
}

func TestAtimePolicy_Relatime_ThresholdExpired(t *testing.T) {
	policy := &AtimePolicy{Mode: AtimeModeRelatime, RelatimeThreshold: time.Hour}
	atime := time.Unix(1_700_000_000, 0)
	existing := Attr{Atime: atime, Mtime: atime, Ctime: atime}
	candidate := atime.Add(2 * time.Hour)
	if !policy.ShouldUpdate(existing, candidate) {
		t.Fatal("relatime must update once threshold elapses")
	}
}

func TestAtimePolicy_Relatime_DebouncesFreshReads(t *testing.T) {
	policy := &AtimePolicy{Mode: AtimeModeRelatime, RelatimeThreshold: 24 * time.Hour}
	atime := time.Unix(1_700_000_000, 0)
	existing := Attr{Atime: atime, Mtime: atime, Ctime: atime}
	candidate := atime.Add(time.Minute)
	if policy.ShouldUpdate(existing, candidate) {
		t.Fatal("relatime must not update within threshold when atime >= mtime/ctime")
	}
}

func TestParseAtimeMode(t *testing.T) {
	cases := map[string]AtimeMode{
		"":          AtimeModeRelatime,
		"relatime":  AtimeModeRelatime,
		"RelAtime":  AtimeModeRelatime,
		"off":       AtimeModeOff,
		"  strict ": AtimeModeStrict,
	}
	for input, expected := range cases {
		got, err := ParseAtimeMode(input)
		if err != nil {
			t.Fatalf("ParseAtimeMode(%q): unexpected error: %v", input, err)
		}
		if got != expected {
			t.Fatalf("ParseAtimeMode(%q): expected %q, got %q", input, expected, got)
		}
	}

	if _, err := ParseAtimeMode("nope"); err == nil {
		t.Fatal("expected ParseAtimeMode to reject unknown mode")
	}
}
