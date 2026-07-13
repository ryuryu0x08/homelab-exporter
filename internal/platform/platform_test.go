package platform

import (
	"testing"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

func TestValidateWindows(t *testing.T) {
	if err := Validate(config.PlatformWindows, "windows"); err != nil {
		t.Fatalf("Validate() error=%v", err)
	}
}

func TestValidateRejectsRuntimeMismatch(t *testing.T) {
	if err := Validate(config.PlatformWindows, "linux"); err == nil {
		t.Fatal("Validate() error=nil, want mismatch error")
	}
}
