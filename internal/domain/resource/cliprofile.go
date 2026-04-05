package resource

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CliBinary string

const (
	BinaryClaude CliBinary = "claude"
)

type CliProfilePreset struct {
	Key          string
	Model        string
	SystemArgs   []string
	SettingsJSON string
	ClaudeJSON   string
}

var CliProfilePresets = []CliProfilePreset{
	{
		Key:          "claude-haiku",
		Model:        "claude-haiku-4-5-20251001",
		SystemArgs:   []string{"--dangerously-skip-permissions"},
		SettingsJSON: `{"skipDangerousModePermissionPrompt":true,"claudeMdExcludes":["~/.claude/**"]}`,
		ClaudeJSON:   `{"hasCompletedOnboarding":true,"projects":{"{{CLIER_MEMBERSPACE}}/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`,
	},
	{
		Key:          "claude-sonnet",
		Model:        "claude-sonnet-4-6",
		SystemArgs:   []string{"--dangerously-skip-permissions"},
		SettingsJSON: `{"skipDangerousModePermissionPrompt":true,"claudeMdExcludes":["~/.claude/**"]}`,
		ClaudeJSON:   `{"hasCompletedOnboarding":true,"projects":{"{{CLIER_MEMBERSPACE}}/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`,
	},
	{
		Key:          "claude-opus",
		Model:        "claude-opus-4-6",
		SystemArgs:   []string{"--dangerously-skip-permissions"},
		SettingsJSON: `{"skipDangerousModePermissionPrompt":true,"claudeMdExcludes":["~/.claude/**"]}`,
		ClaudeJSON:   `{"hasCompletedOnboarding":true,"projects":{"{{CLIER_MEMBERSPACE}}/project":{"hasTrustDialogAccepted":true,"hasCompletedProjectOnboarding":true}}}`,
	},
}

type CliProfile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Model        string    `json:"model"`
	Binary       CliBinary `json:"binary"`
	SystemArgs   []string  `json:"system_args"`
	CustomArgs   []string  `json:"custom_args"`
	SettingsJSON string    `json:"settings_json"`
	ClaudeJSON   string    `json:"claude_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
		ID:           uuid.NewString(),
		Name:         name,
		Model:        preset.Model,
		Binary:       BinaryClaude,
		SystemArgs:   append([]string{}, preset.SystemArgs...),
		CustomArgs:   customArgs,
		SettingsJSON: preset.SettingsJSON,
		ClaudeJSON:   preset.ClaudeJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// validateRawFields validates and normalises shared raw-profile inputs.
func validateRawFields(name, model string, binary CliBinary, systemArgs, customArgs []string) (
	string, string, []string, []string, error,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", nil, nil, errors.New("cli profile name must not be empty")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", "", nil, nil, errors.New("cli profile model must not be empty")
	}
	if binary != BinaryClaude {
		return "", "", nil, nil, fmt.Errorf("invalid binary: %s (must be claude)", binary)
	}
	if systemArgs == nil {
		systemArgs = []string{}
	}
	if customArgs == nil {
		customArgs = []string{}
	}
	return name, model, append([]string{}, systemArgs...), append([]string{}, customArgs...), nil
}

// NewCliProfileRaw creates a CliProfile from explicit values (no preset lookup).
// Used by import to recreate profiles from exported data.
func NewCliProfileRaw(name, model string, binary CliBinary, systemArgs, customArgs []string, settingsJSON, claudeJSON string) (*CliProfile, error) {
	name, model, systemArgs, customArgs, err := validateRawFields(name, model, binary, systemArgs, customArgs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &CliProfile{
		ID:           uuid.NewString(),
		Name:         name,
		Model:        model,
		Binary:       binary,
		SystemArgs:   systemArgs,
		CustomArgs:   customArgs,
		SettingsJSON: settingsJSON,
		ClaudeJSON:   claudeJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
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
func (p *CliProfile) UpdateRaw(name, model string, binary CliBinary, systemArgs, customArgs []string, settingsJSON, claudeJSON string) error {
	name, model, systemArgs, customArgs, err := validateRawFields(name, model, binary, systemArgs, customArgs)
	if err != nil {
		return err
	}

	p.Name = name
	p.Model = model
	p.Binary = binary
	p.SystemArgs = systemArgs
	p.CustomArgs = customArgs
	p.SettingsJSON = settingsJSON
	p.ClaudeJSON = claudeJSON
	p.UpdatedAt = time.Now()
	return nil
}
