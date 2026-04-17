package cmd

import (
	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

// requireOneArg returns a cobra.PositionalArgs validator that enforces
// exactly one argument. Failures are returned as domain.Fault values so
// the central presenter renders the message — cobra's default
// "accepts N arg(s), received M" never reaches the user.
func requireOneArg(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		switch {
		case len(args) == 1:
			return nil
		case len(args) > 1:
			return &domain.Fault{
				Kind:    domain.KindTooManyArgs,
				Subject: map[string]string{"usage": usage},
			}
		default:
			return &domain.Fault{
				Kind:    domain.KindMissingArgument,
				Subject: map[string]string{"usage": usage},
			}
		}
	}
}

// requireMaxArgs returns a validator allowing 0..max args.
func requireMaxArgs(max int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) <= max {
			return nil
		}
		return &domain.Fault{
			Kind:    domain.KindTooManyArgs,
			Subject: map[string]string{"usage": usage},
		}
	}
}
