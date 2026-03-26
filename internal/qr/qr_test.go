package qr

import (
	"testing"
)

func TestGeneratePNG(t *testing.T) {
	png, err := GeneratePNG("test content", 256)
	if err != nil {
		t.Fatalf("GeneratePNG: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("PNG should not be empty")
	}
	// Check PNG magic bytes
	if png[0] != 0x89 || png[1] != 0x50 || png[2] != 0x4E || png[3] != 0x47 {
		t.Fatal("output doesn't look like a PNG")
	}
}

func TestGenerateTerminal(t *testing.T) {
	out, err := GenerateTerminal("test content")
	if err != nil {
		t.Fatalf("GenerateTerminal: %v", err)
	}
	if out == "" {
		t.Fatal("terminal output should not be empty")
	}
	if len(out) < 100 {
		t.Fatal("terminal QR output seems too small")
	}
}
