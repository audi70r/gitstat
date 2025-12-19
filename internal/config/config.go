package config

import (
	"time"
)

// Config holds application configuration
type Config struct {
	// Repository settings
	RepoPath  string   // Primary repo (for backwards compatibility)
	RepoPaths []string // Multiple repositories
	Since     time.Time
	Until     time.Time

	// Display settings
	Timezone      *time.Location
	TimeFormat24h bool

	// Limits
	MaxAuthors int
	MaxFiles   int

	// Timeline settings
	SparklineWidth int
	RollingWindow  int // Days for rolling average

	// Hotspot thresholds
	HotspotChurnThreshold  float64
	HotspotAuthorThreshold int
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Timezone:               time.Local,
		TimeFormat24h:          true,
		MaxAuthors:             20,
		MaxFiles:               30,
		SparklineWidth:         52,
		RollingWindow:          7,
		HotspotChurnThreshold:  0.7,
		HotspotAuthorThreshold: 3,
	}
}
