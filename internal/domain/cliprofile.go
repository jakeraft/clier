package domain

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CliBinary string

const (
	BinaryClaude CliBinary = "claude"
	BinaryCodex  CliBinary = "codex"
)

type DotConfig map[string]any

type CliProfilePreset struct {
	Key        string
	Binary     CliBinary
	Model      string
	SystemArgs []string
	DotConfig  DotConfig
}

var CliProfilePresets = []CliProfilePreset{
	{
		Key:        "claude-haiku",
		Binary:     BinaryClaude,
		Model:      "claude-haiku-4-5-20251001",
		SystemArgs: []string{"--dangerously-skip-permissions"},
		DotConfig: DotConfig{
			"skipDangerousModePermissionPrompt": true,
			"claudeMdExcludes":                  []string{"~/.claude/**"},
		},
	},
	{
		Key:        "claude-sonnet",
		Binary:     BinaryClaude,
		Model:      "claude-sonnet-4-6",
		SystemArgs: []string{"--dangerously-skip-permissions"},
		DotConfig: DotConfig{
			"skipDangerousModePermissionPrompt": true,
			"claudeMdExcludes":                  []string{"~/.claude/**"},
		},
	},
	{
		Key:        "claude-opus",
		Binary:     BinaryClaude,
		Model:      "claude-opus-4-6",
		SystemArgs: []string{"--dangerously-skip-permissions"},
		DotConfig: DotConfig{
			"skipDangerousModePermissionPrompt": true,
			"claudeMdExcludes":                  []string{"~/.claude/**"},
		},
	},
	{
		Key:        "codex-5.4",
		Binary:     BinaryCodex,
		Model:      "gpt-5.4",
		SystemArgs: []string{},
		DotConfig: DotConfig{
			"sandbox_mode": "danger-full-access",
			"notice": map[string]any{
				"model_migrations": map[string]any{},
			},
		},
	},
	{
		Key:        "codex-mini",
		Binary:     BinaryCodex,
		Model:      "gpt-5.1-codex-mini",
		SystemArgs: []string{},
		DotConfig: DotConfig{
			"sandbox_mode": "danger-full-access",
			"notice": map[string]any{
				"model_migrations": map[string]any{
					"gpt-5.1-codex-mini": "gpt-5.4",
				},
			},
		},
	},
}

type CliProfile struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	Binary     CliBinary `json:"binary"`
	SystemArgs []string  `json:"system_args"`
	CustomArgs []string  `json:"custom_args"`
	DotConfig  DotConfig `json:"dot_config"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewCliProfile(name, presetKey string, customArgs []string) (*CliProfile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("cli profile name must not be empty")
	}

	preset, err := ResolvePreset(presetKey)
	if err != nil {
		return nil, err
	}

	if customArgs == nil {
		customArgs = []string{}
	}

	now := time.Now()
	return &CliProfile{
		ID:         uuid.NewString(),
		Name:       name,
		Model:      preset.Model,
		Binary:     preset.Binary,
		SystemArgs: append([]string{}, preset.SystemArgs...),
		CustomArgs: customArgs,
		DotConfig:  copyDotConfig(preset.DotConfig),
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func ResolvePreset(key string) (CliProfilePreset, error) {
	for _, p := range CliProfilePresets {
		if p.Key == key {
			return p, nil
		}
	}
	return CliProfilePreset{}, fmt.Errorf("unknown preset: %s", key)
}

var dateSuffixRe = regexp.MustCompile(`-\d{8}$`)

func StripDateSuffix(modelID string) string {
	return dateSuffixRe.ReplaceAllString(modelID, "")
}

func (p *CliProfile) MatchesRawID(rawID string) bool {
	return StripDateSuffix(rawID) == p.Model
}

func (p *CliProfile) Update(name *string, customArgs *[]string) error {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return errors.New("cli profile name must not be empty")
		}
		p.Name = trimmed
	}
	if customArgs != nil {
		p.CustomArgs = *customArgs
	}
	p.UpdatedAt = time.Now()
	return nil
}

func copyDotConfig(src DotConfig) DotConfig {
	if src == nil {
		return nil
	}
	dst := make(DotConfig, len(src))
	maps.Copy(dst, src)
	return dst
}
