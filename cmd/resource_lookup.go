package cmd

import (
	"errors"
	"fmt"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func resolveServerResourceKind(client *api.Client, owner, name string) (string, error) {
	r, err := client.GetResource(owner, name)
	if err != nil {
		if isNotFound(err) {
			return "", fmt.Errorf("resource %s/%s was not found", owner, name)
		}
		return "", err
	}

	switch api.ResourceKind(r.Kind) {
	case api.KindTeam:
		return resourceKindTeam, nil
	case api.KindMember:
		return resourceKindMember, nil
	case api.KindSkill:
		return resourceKindSkill, nil
	case api.KindClaudeMd:
		return resourceKindClaudeMd, nil
	case api.KindClaudeSettings:
		return resourceKindClaudeSettings, nil
	default:
		return r.Kind, nil
	}
}

func isNotFound(err error) bool {
	var apiErr *api.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}
