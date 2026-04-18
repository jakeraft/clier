package cmd

import (
	"errors"
	"os"

	"github.com/jakeraft/clier/internal/domain"
)

// errNoWorkingCopy returns a uniform "not cloned yet" Fault used by
// status / pull / push / run start so the user always gets the same
// remediation hint regardless of which command they tried.
func errNoWorkingCopy(owner, name, base string) error {
	return &domain.Fault{
		Kind: domain.KindWorkingCopyMissing,
		Subject: map[string]string{
			"path":  base,
			"owner": owner,
			"name":  name,
		},
	}
}

// classifyWorkingCopyError wraps the raw os.ErrNotExist from manifest
// loading into the friendly errNoWorkingCopy error. Other errors pass
// through untouched. Uses errors.Is so wrapped errors (e.g.,
// "read manifest: %w") still match.
func classifyWorkingCopyError(owner, name, base string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return errNoWorkingCopy(owner, name, base)
	}
	return err
}
