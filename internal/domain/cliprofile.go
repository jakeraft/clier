package domain

import (
	"encoding/json"
	"errors"
	"fmt"
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

// NewCliProfileRaw creates a CliProfile from explicit values (no preset lookup).
// Used by import to recreate profiles from exported data.
func NewCliProfileRaw(name, model string, binary CliBinary, systemArgs, customArgs []string, dotConfig DotConfig) (*CliProfile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("cli profile name must not be empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, errors.New("cli profile model must not be empty")
	}

	switch binary {
	case BinaryClaude, BinaryCodex:
	default:
		return nil, fmt.Errorf("invalid binary: %s (must be claude or codex)", binary)
	}

	if systemArgs == nil {
		systemArgs = []string{}
	}
	if customArgs == nil {
		customArgs = []string{}
	}

	now := time.Now()
	return &CliProfile{
		ID:         uuid.NewString(),
		Name:       name,
		Model:      model,
		Binary:     binary,
		SystemArgs: append([]string{}, systemArgs...),
		CustomArgs: append([]string{}, customArgs...),
		DotConfig:  copyDotConfig(dotConfig),
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

// UpdateRaw replaces all mutable fields with validated, deep-copied values.
// Used by import to fully overwrite an existing profile from exported data.
func (p *CliProfile) UpdateRaw(name, model string, binary CliBinary, systemArgs, customArgs []string, dotConfig DotConfig) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("cli profile name must not be empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("cli profile model must not be empty")
	}
	switch binary {
	case BinaryClaude, BinaryCodex:
	default:
		return fmt.Errorf("invalid binary: %s (must be claude or codex)", binary)
	}
	if systemArgs == nil {
		systemArgs = []string{}
	}
	if customArgs == nil {
		customArgs = []string{}
	}

	p.Name = name
	p.Model = model
	p.Binary = binary
	p.SystemArgs = append([]string{}, systemArgs...)
	p.CustomArgs = append([]string{}, customArgs...)
	p.DotConfig = copyDotConfig(dotConfig)
	p.UpdatedAt = time.Now()
	return nil
}

func copyDotConfig(src DotConfig) DotConfig {
	if src == nil {
		return nil
	}
	data, _ := json.Marshal(src)
	var dst DotConfig
	_ = json.Unmarshal(data, &dst)
	return dst
}
