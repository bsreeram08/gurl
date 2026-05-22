package commands

import (
	"context"
	"strings"
	"testing"
)

func TestAuthCommandListShowsBuiltInTypes(t *testing.T) {
	cmd := AuthCommand()

	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"auth", "list"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, authType := range []string{"apikey", "awsv4", "basic", "bearer", "digest", "ntlm", "oauth1", "oauth2"} {
		if !strings.Contains(output, authType) {
			t.Fatalf("expected auth list to include %q, got %q", authType, output)
		}
	}
}

func TestAuthCommandInfoShowsParameterMetadata(t *testing.T) {
	cmd := AuthCommand()

	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"auth", "info", "oauth2"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{
		"oauth2",
		"client_id",
		"required",
		"client_secret",
		"secret",
		"token_url",
		"OAuth 2.0 token endpoint URL",
		"flow",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected auth info to include %q, got %q", want, output)
		}
	}
}

func TestAuthCommandInfoShowsDefaults(t *testing.T) {
	cmd := AuthCommand()

	output := captureStdout(t, func() {
		if err := cmd.Run(context.Background(), []string{"auth", "info", "digest"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"realm", "default-realm", "algorithm", "MD5"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected digest auth info to include %q, got %q", want, output)
		}
	}
}

func TestAuthCommandInfoRejectsUnknownType(t *testing.T) {
	cmd := AuthCommand()

	err := cmd.Run(context.Background(), []string{"auth", "info", "madeup"})
	if err == nil {
		t.Fatal("expected unknown auth type error")
	}
	if !strings.Contains(err.Error(), "unknown auth type") {
		t.Fatalf("expected unknown auth type error, got %v", err)
	}
}
