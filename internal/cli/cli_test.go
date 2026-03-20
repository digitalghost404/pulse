package cli

import (
	"testing"
	"time"
)

func TestParseSinceDuration_Days(t *testing.T) {
	d, err := parseSinceDuration("7d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 7*24*time.Hour {
		t.Errorf("expected 168h, got %v", d)
	}
}

func TestParseSinceDuration_Hours(t *testing.T) {
	d, err := parseSinceDuration("24h")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 24*time.Hour {
		t.Errorf("expected 24h, got %v", d)
	}
}

func TestParseSinceDuration_Minutes(t *testing.T) {
	d, err := parseSinceDuration("30m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 30*time.Minute {
		t.Errorf("expected 30m, got %v", d)
	}
}

func TestParseSinceDuration_Invalid(t *testing.T) {
	_, err := parseSinceDuration("abc")
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestParseSinceDuration_OneDayEdge(t *testing.T) {
	d, err := parseSinceDuration("1d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 24*time.Hour {
		t.Errorf("expected 24h, got %v", d)
	}
}
