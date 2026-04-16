package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
)

func parseOptionalResourceRefRequest(raw string) (*api.ResourceRefRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	at := strings.LastIndex(raw, "@")
	if at <= 0 || at == len(raw)-1 {
		return nil, fmt.Errorf("invalid resource ref %q: want <owner/name>@<version>", raw)
	}
	owner, name, err := splitResourceID(strings.TrimSpace(raw[:at]))
	if err != nil {
		return nil, fmt.Errorf("invalid resource ref %q: %w", raw, err)
	}
	version, err := strconv.Atoi(strings.TrimSpace(raw[at+1:]))
	if err != nil {
		return nil, fmt.Errorf("invalid resource version in %q: %w", raw, err)
	}
	return &api.ResourceRefRequest{Owner: owner, Name: name, Version: version}, nil
}

func parseResourceRefSlice(specs []string) ([]api.ResourceRefRequest, error) {
	refs := make([]api.ResourceRefRequest, 0, len(specs))
	for _, spec := range specs {
		ref, err := parseOptionalResourceRefRequest(spec)
		if err != nil {
			return nil, err
		}
		if ref == nil {
			return nil, errors.New("resource ref must not be empty")
		}
		refs = append(refs, *ref)
	}
	return refs, nil
}

func parseChildRefSpecs(specs []string) ([]api.ChildRefRequest, error) {
	children := make([]api.ChildRefRequest, 0, len(specs))
	for _, spec := range specs {
		ref, err := parseOptionalResourceRefRequest(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid --child %q: %w", spec, err)
		}
		if ref == nil {
			return nil, fmt.Errorf("invalid --child %q, child ref must not be empty", spec)
		}
		children = append(children, api.ChildRefRequest{
			Owner: ref.Owner, Name: ref.Name, ChildVersion: ref.Version,
		})
	}
	return children, nil
}
