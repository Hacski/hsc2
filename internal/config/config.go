package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ListenerConfig struct {
	Type     string            `json:"type"`
	Addr     string            `json:"addr"`
	CertFile string            `json:"cert_file,omitempty"`
	KeyFile  string            `json:"key_file,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

type BeaconProfileConfig struct {
	Name          string  `json:"name"`
	SleepSeconds  int     `json:"sleep_seconds"`
	JitterPercent float64 `json:"jitter_percent"`
	UserAgent     string  `json:"user_agent,omitempty"`
	URIs          []string `json:"uris,omitempty"`
}

type PayloadConfig struct {
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
	Format        string   `json:"format"`
	ObfuscatePipe []string `json:"obfuscate_pipe,omitempty"`
	BeaconProfile string   `json:"beacon_profile"`
}

type TeamServerConfig struct {
	ListenAddr  string `json:"listen_addr"`
	DBDir       string `json:"db_dir"`
	CertDir     string `json:"cert_dir"`
	LogDir      string `json:"log_dir"`
	MaxOperators int   `json:"max_operators"`
}

type EngagementConfig struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Server      TeamServerConfig      `json:"server"`
	Listeners   []ListenerConfig      `json:"listeners"`
	Profiles    []BeaconProfileConfig `json:"profiles"`
	Payloads    []PayloadConfig       `json:"payloads,omitempty"`
}

var ErrInvalidConfig = errors.New("invalid config")

func LoadJSON(path string) (*EngagementConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg EngagementConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *EngagementConfig) SaveJSON(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func (c *EngagementConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: engagement name is required", ErrInvalidConfig)
	}
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("%w: server.listen_addr is required", ErrInvalidConfig)
	}
	for i, l := range c.Listeners {
		if l.Type == "" {
			return fmt.Errorf("%w: listener[%d].type is required", ErrInvalidConfig, i)
		}
		if l.Addr == "" {
			return fmt.Errorf("%w: listener[%d].addr is required", ErrInvalidConfig, i)
		}
	}
	for i, p := range c.Profiles {
		if p.Name == "" {
			return fmt.Errorf("%w: profile[%d].name is required", ErrInvalidConfig, i)
		}
		if p.SleepSeconds <= 0 {
			return fmt.Errorf("%w: profile[%d].sleep_seconds must be positive", ErrInvalidConfig, i)
		}
		if p.JitterPercent < 0 || p.JitterPercent > 100 {
			return fmt.Errorf("%w: profile[%d].jitter_percent must be 0..100", ErrInvalidConfig, i)
		}
	}
	return nil
}

func DefaultConfig() *EngagementConfig {
	return &EngagementConfig{
		Name:    "default-engagement",
		Version: "1.0.0",
		Server: TeamServerConfig{
			ListenAddr:   "0.0.0.0:8443",
			DBDir:        "./data/db",
			CertDir:      "./data/certs",
			LogDir:       "./data/logs",
			MaxOperators: 10,
		},
		Listeners: []ListenerConfig{
			{Type: "https", Addr: "0.0.0.0:443"},
			{Type: "dns", Addr: "0.0.0.0:53", Options: map[string]string{"domain": "c2.example.com"}},
		},
		Profiles: []BeaconProfileConfig{
			{
				Name:          "slow-burn",
				SleepSeconds:  300,
				JitterPercent: 25,
				UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			},
		},
	}
}
