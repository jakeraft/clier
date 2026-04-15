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
		return string(api.KindTeam), nil
	case api.KindMember:
		return string(api.KindMember), nil
	case api.KindSkill:
		return string(api.KindSkill), nil
	case api.KindClaudeMd:
		return string(api.KindClaudeMd), nil
	case api.KindClaudeSettings:
		return string(api.KindClaudeSettings), nil
	case api.KindCodexMd:
		return string(api.KindCodexMd), nil
	case api.KindCodexSettings:
		return string(api.KindCodexSettings), nil
	default:
		return r.Kind, nil
	}
}

func isNotFound(err error) bool {
	var apiErr *api.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}
