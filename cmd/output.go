// Package cmd implements the clier CLI commands.
//
// # Output conventions
//
// All commands write a single JSON object to stdout. Agents can parse
// the output with `jq` or any JSON decoder.
//
//   - Field names are snake_case across every command (e.g. run_id,
//     started_at, owner_name).
//   - Empty collections are rendered as [] — never null. A missing list
//     means "no items," not "value unknown."
//   - Timestamps from the server are RFC3339 in UTC (suffix Z). Runtime
//     events generated locally (run notes, plan files) keep the host's
//     local offset so users can correlate with their own clock.
//
// Errors are written to stderr in the form `Error: <message>` and the
// process exits with code 1. Messages are user-facing — they avoid
// raw HTTP status codes and never leak terminal-multiplexer internals.
package cmd
