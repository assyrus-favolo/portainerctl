package client

import (
	"github.com/portainer/portainerctl/internal/config"
)

// MustClient loads config and returns a ready Client, or panics with a user-facing error.
func MustClient() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	ctx, err := cfg.Current()
	if err != nil {
		return nil, err
	}
	return New(ctx), nil
}
