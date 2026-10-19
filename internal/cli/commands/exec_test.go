package commands

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestExecCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}

	tests := []struct {
		name     string
		active   string
		args     []string
		wantErr  bool
		wantCode int // expected cli.ExitCoder code; 0 means no ExitCoder
	}{
		{name: "injects env var", args: []string{"--env", "dev", "--", "sh", "-c", `test "$FOO" = bar`}},
		{name: "uses active env", active: "dev", args: []string{"--", "sh", "-c", `test "$FOO" = bar`}},
		{name: "works without double dash", args: []string{"--env", "dev", "sh", "-c", `test "$FOO" = bar`}},
		{name: "var override", args: []string{"--env", "dev", "--var", "FOO=baz", "--", "sh", "-c", `test "$FOO" = baz`}},
		{name: "propagates exit code", args: []string{"--env", "dev", "--", "sh", "-c", "exit 7"}, wantErr: true, wantCode: 7},
		{name: "env var mismatch fails", args: []string{"--env", "dev", "--", "sh", "-c", `test "$FOO" = nope`}, wantErr: true, wantCode: 1},
		{name: "no command", args: []string{"--env", "dev"}, wantErr: true},
		{name: "no env selected", args: []string{"--", "true"}, wantErr: true},
		{name: "unknown env", args: []string{"--env", "missing", "--", "true"}, wantErr: true},
		{name: "command not found", args: []string{"--env", "dev", "--", "gurl-definitely-not-a-command"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newMockEnvDB()
			helperCreateTestEnv(db, "dev", map[string]string{"FOO": "bar"})
			db.activeEnv = tt.active

			cmd := ExecCommand(db)
			cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}

			err := cmd.Run(context.Background(), append([]string{"exec"}, tt.args...))
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			var exitCoder cli.ExitCoder
			isExit := errors.As(err, &exitCoder)
			if tt.wantCode == 0 && isExit {
				t.Errorf("unexpected exit coder with code %d", exitCoder.ExitCode())
			}
			if tt.wantCode != 0 {
				if !isExit {
					t.Fatalf("expected cli.ExitCoder, got %v", err)
				}
				if exitCoder.ExitCode() != tt.wantCode {
					t.Errorf("exit code = %d, want %d", exitCoder.ExitCode(), tt.wantCode)
				}
			}
		})
	}
}
