package cmd

import "errors"

func errNotInWorkingCopy() error {
	return errors.New("no local clone found in the current directory or its ancestors")
}
