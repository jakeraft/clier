package cmd

import (
	"strconv"
	"strings"

	"github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

func parseOptionalResourceRefRequest(raw string) (*api.ResourceRefRequest, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	at := strings.LastIndex(raw, "@")
	if at <= 0 || at == len(raw)-1 {
		return nil, &domain.Fault{
			Kind:    domain.KindInvalidResourceRef,
			Subject: map[string]string{"ref": raw},
		}
	}
	owner, name, err := splitResourceID(strings.TrimSpace(raw[:at]))
	if err != nil {
		return nil, &domain.Fault{
			Kind:    domain.KindInvalidResourceRef,
			Subject: map[string]string{"ref": raw},
			Cause:   err,
		}
	}
	version, err := strconv.Atoi(strings.TrimSpace(raw[at+1:]))
	if err != nil {
		return nil, &domain.Fault{
			Kind:    domain.KindInvalidResourceRef,
			Subject: map[string]string{"ref": raw},
			Cause:   err,
		}
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
			return nil, &domain.Fault{
				Kind:    domain.KindInvalidArgument,
				Subject: map[string]string{"detail": "resource ref must not be empty"},
			}
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
			return nil, &domain.Fault{
				Kind:    domain.KindInvalidArgument,
				Subject: map[string]string{"detail": "invalid --child " + quote(spec)},
				Cause:   err,
			}
		}
		if ref == nil {
			return nil, &domain.Fault{
				Kind:    domain.KindInvalidArgument,
				Subject: map[string]string{"detail": "invalid --child " + quote(spec) + ": child ref must not be empty"},
			}
		}
		children = append(children, api.ChildRefRequest{
			Owner: ref.Owner, Name: ref.Name, ChildVersion: ref.Version,
		})
	}
	return children, nil
}

func quote(s string) string { return `"` + s + `"` }
