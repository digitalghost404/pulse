package writer_test

import (
	"testing"

	"github.com/xcoleman/pulse/internal/writer"
)

func TestWriterRegistry_StdoutRegistered(t *testing.T) {
	w, ok := writer.Get("stdout")
	if !ok {
		t.Fatal("expected stdout writer to be registered")
	}
	if w.Name() != "stdout" {
		t.Errorf("expected name 'stdout', got %s", w.Name())
	}
}

func TestWriterRegistry_ObsidianRegistered(t *testing.T) {
	w, ok := writer.Get("obsidian")
	if !ok {
		t.Fatal("expected obsidian writer to be registered")
	}
	if w.Name() != "obsidian" {
		t.Errorf("expected name 'obsidian', got %s", w.Name())
	}
}

func TestWriterRegistry_All(t *testing.T) {
	all := writer.All()
	if len(all) < 2 {
		t.Errorf("expected at least 2 writers, got %d", len(all))
	}

	names := make(map[string]bool)
	for _, w := range all {
		names[w.Name()] = true
	}
	if !names["stdout"] {
		t.Error("expected stdout in All()")
	}
	if !names["obsidian"] {
		t.Error("expected obsidian in All()")
	}
}

func TestWriterRegistry_NotFound(t *testing.T) {
	_, ok := writer.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent writer to not be found")
	}
}
