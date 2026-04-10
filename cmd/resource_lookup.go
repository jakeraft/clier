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

	switch {
	case teamExists && memberExists:
		return "", fmt.Errorf("resource %s/%s is ambiguous across team and member definitions", owner, name)
	case teamExists:
		return resourceKindTeam, nil
	case memberExists:
		return resourceKindMember, nil
	default:
		return "", fmt.Errorf("resource %s/%s was not found", owner, name)
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

func isNotFound(err error) bool {
	var apiErr *api.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}
