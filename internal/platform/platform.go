package platform

import (
	"fmt"

	"github.com/ryuryu0x08/homelab-exporter/internal/config"
)

// Validate checks that the configured platform matches the running binary.
func Validate(configured config.Platform, runtimeName string) error {
	if string(configured) != runtimeName {
		return fmt.Errorf("configured platform %q does not match runtime %q", configured, runtimeName)
	}
	return nil
}
