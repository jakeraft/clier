package cmd

import (
	"errors"
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func resolveServerResourceKind(client *api.Client, owner, name string) (string, error) {
	teamExists, err := teamExistsOnServer(client, owner, name)
	if err != nil {
		return "", err
	}
	memberExists, err := memberExistsOnServer(client, owner, name)
	if err != nil {
		return "", err
	}
	skillExists, err := skillExistsOnServer(client, owner, name)
	if err != nil {
		return "", err
	}
	claudeMdExists, err := claudeMdExistsOnServer(client, owner, name)
	if err != nil {
		return "", err
	}
	claudeSettingsExists, err := claudeSettingsExistsOnServer(client, owner, name)
	if err != nil {
		return "", err
	}

	matches := make([]string, 0, 5)
	if teamExists {
		matches = append(matches, resourceKindTeam)
	}
	if memberExists {
		matches = append(matches, resourceKindMember)
	}
	if skillExists {
		matches = append(matches, resourceKindSkill)
	}
	if claudeMdExists {
		matches = append(matches, resourceKindClaudeMd)
	}
	if claudeSettingsExists {
		matches = append(matches, resourceKindClaudeSettings)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("resource %s/%s was not found", owner, name)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("resource %s/%s is ambiguous across %v", owner, name, matches)
	}
}

func teamExistsOnServer(client *api.Client, owner, name string) (bool, error) {
	_, err := client.GetTeam(owner, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func memberExistsOnServer(client *api.Client, owner, name string) (bool, error) {
	_, err := client.GetMember(owner, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func skillExistsOnServer(client *api.Client, owner, name string) (bool, error) {
	_, err := client.GetSkill(owner, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func claudeMdExistsOnServer(client *api.Client, owner, name string) (bool, error) {
	_, err := client.GetClaudeMd(owner, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func claudeSettingsExistsOnServer(client *api.Client, owner, name string) (bool, error) {
	_, err := client.GetClaudeSettings(owner, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

func isNotFound(err error) bool {
	var apiErr *api.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}
