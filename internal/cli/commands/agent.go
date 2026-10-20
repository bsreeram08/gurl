package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

// Agent mode makes gurl drivable by AI agents: every command that supports
// --json prints JSON, nothing prompts, --reveal and the clipboard are
// refused, and secrets are also masked inside response bodies.
//
// Non-interactive mode (--no-input, implied by agent mode and by --json)
// only stops prompts: a command that would ask fails with an error naming
// the flag or argument to pass instead.
const (
	AgentEnvVar   = "GURL_AGENT"
	NoInputEnvVar = "GURL_NO_INPUT"

	agentFlag   = "agent"
	noInputFlag = "no-input"
)

// GlobalFlags are the root flags shared by every subcommand.
func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  agentFlag,
			Usage: "JSON output, no prompts, strict masking (or " + AgentEnvVar + "=1, or run as agentgurl)",
		},
		&cli.BoolFlag{
			Name:  noInputFlag,
			Usage: "Never prompt; fail naming what to pass (or " + NoInputEnvVar + "=1)",
		},
	}
}

// envEnabled reports whether the environment variable name is set to a true
// value (1, true, yes, on).
func envEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// agentMode reports whether c runs in agent mode.
func agentMode(c *cli.Command) bool {
	return envEnabled(AgentEnvVar) || c.Bool(agentFlag)
}

// AgentRequested reports, before flags are parsed, whether args or the
// environment ask for agent mode.
func AgentRequested(args []string) bool {
	if envEnabled(AgentEnvVar) {
		return true
	}
	for _, arg := range args {
		switch arg {
		case "--":
			return false
		case "--agent", "-agent", "--agent=true", "--agent=1":
			return true
		}
	}
	return false
}

// jsonOutput reports whether c prints JSON: --json, or agent mode.
func jsonOutput(c *cli.Command) bool {
	return c.Bool("json") || agentMode(c)
}

// noInput reports whether c must never prompt.
func noInput(c *cli.Command) bool {
	return jsonOutput(c) || c.Bool(noInputFlag) || envEnabled(NoInputEnvVar)
}

// canPrompt is the one decision whether c may ask the user something:
// prompts are allowed unless non-interactive mode is on, and only when
// terminal reports an interactive stdin.
func canPrompt(c *cli.Command, terminal func() bool) bool {
	return !noInput(c) && terminal()
}

// rejectReveal refuses --reveal in agent mode.
func rejectReveal(c *cli.Command) error {
	if agentMode(c) && c.Bool("reveal") {
		return cli.Exit("--reveal is not allowed in agent mode", 2)
	}
	return nil
}

// reportedError is an error already reported on stdout as JSON (an
// execution object or an error object), so it is not reported again.
type reportedError struct{ error }

func (e reportedError) Unwrap() error { return e.error }

// reportedExit keeps the exit code of a reported cli.ExitCoder.
type reportedExit struct {
	reportedError
	code int
}

func (e reportedExit) ExitCode() int { return e.code }

// markReported marks err as reported; nil stays nil.
func markReported(err error) error {
	if err == nil || isReported(err) {
		return err
	}
	var exit cli.ExitCoder
	if errors.As(err, &exit) {
		return reportedExit{reportedError{err}, exit.ExitCode()}
	}
	return reportedError{err}
}

func isReported(err error) bool {
	var r reportedError
	var x reportedExit
	return errors.As(err, &r) || errors.As(err, &x)
}

// jsonError is the object printed for an error in JSON mode that is not
// part of an execution: unknown request or environment, bad flags, ...
type jsonError struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// reportJSONError prints err as a JSON error object and returns it marked
// as reported.
func reportJSONError(w io.Writer, err error) error {
	if werr := writeJSONLine(w, jsonError{Error: err.Error()}); werr != nil {
		return fmt.Errorf("%w (writing JSON error: %v)", err, werr)
	}
	return markReported(err)
}

// writeJSONLine writes v as one line of JSON. <, > and & are not escaped:
// the output is for programs, not HTML.
func writeJSONLine(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// writeJSON writes v as indented JSON, for commands that print one document.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonAction wraps a command action so that in JSON mode an error the action
// did not report itself is printed on stdout as {"ok":false,"error":"..."}.
// The error is still returned, so it also reaches stderr and the exit code.
func jsonAction(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, c *cli.Command) error {
		err := action(ctx, c)
		if err == nil || isReported(err) || !jsonOutput(c) {
			return err
		}
		return reportJSONError(os.Stdout, err)
	}
}

// ExitErrHandler handles errors returned by the root command. In agent mode
// (agent: requested before parsing, see AgentRequested) an error no command
// reported, such as a usage error or a failure in a command without JSON
// output, is printed as a JSON error object first. Exit codes are as
// without the handler.
func ExitErrHandler(agent bool) cli.ExitErrHandlerFunc {
	done := false
	return func(ctx context.Context, c *cli.Command, err error) {
		if err != nil && !done && !isReported(err) && (agent || agentMode(c)) {
			done = true
			if werr := writeJSONLine(os.Stdout, jsonError{Error: err.Error()}); werr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write JSON error: %v\n", werr)
			}
		}
		cli.HandleExitCoder(err)
	}
}

// QuietUsageErrors stops cmd and its subcommands from printing help on a
// usage error, so stdout only carries JSON in agent mode.
func QuietUsageErrors(cmd *cli.Command) {
	cmd.OnUsageError = func(ctx context.Context, c *cli.Command, err error, isSubcommand bool) error {
		return err
	}
	for _, sub := range cmd.Commands {
		QuietUsageErrors(sub)
	}
}
