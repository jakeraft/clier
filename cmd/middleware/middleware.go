// Package middleware decorates cobra RunE functions with cross-cutting
// concerns (panic recovery, error translation, telemetry). It mirrors
// the chain-of-responsibility pattern common in HTTP backends:
//
//	mw := middleware.Chain(Recover, Translate)
//	middleware.Apply(rootCmd, mw)
//
// Apply walks the command tree once at startup and wraps every RunE
// in place. New commands are automatically covered without per-command
// boilerplate.
package middleware

import (
	"fmt"

	"github.com/jakeraft/clier/internal/domain"
	"github.com/spf13/cobra"
)

// RunE is the cobra command-run signature.
type RunE = func(*cobra.Command, []string) error

// Middleware wraps a RunE to add behavior before/after invocation.
type Middleware func(RunE) RunE

// Chain composes middlewares so the first listed runs outermost.
func Chain(mws ...Middleware) Middleware {
	return func(final RunE) RunE {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}

// Apply walks the command tree and decorates every RunE with mw.
// Commands without a RunE (parent groups) are left untouched.
//
// Apply also forces SilenceUsage and SilenceErrors on every node so the
// CLI never bypasses the central presenter with cobra's default usage
// banner or "Error:" prefix.
func Apply(root *cobra.Command, mw Middleware) {
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.SilenceUsage = true
		c.SilenceErrors = true
		c.SetFlagErrorFunc(flagErrorFunc)
		if c.RunE != nil {
			c.RunE = mw(c.RunE)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// flagErrorFunc converts cobra flag-parsing failures into Faults so the
// presenter renders them through the catalog instead of cobra's
// "Error: unknown flag --foo" string.
func flagErrorFunc(c *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	return &domain.Fault{
		Kind:    domain.KindInvalidArgument,
		Subject: map[string]string{"detail": err.Error()},
		Cause:   err,
	}
}

// Recover converts panics into KindInternal Faults so the CLI never
// terminates with a stack trace on stderr.
func Recover(next RunE) RunE {
	return func(c *cobra.Command, args []string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = &domain.Fault{
					Kind:    domain.KindInternal,
					Subject: map[string]string{"detail": fmt.Sprintf("panic: %v", r)},
				}
			}
		}()
		return next(c, args)
	}
}
