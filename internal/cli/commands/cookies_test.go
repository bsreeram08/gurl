package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sreeram/gurl/internal/cookies"
	"github.com/sreeram/gurl/internal/redact"
	"github.com/urfave/cli/v3"
)

const cookieTestSecret = "sess-7f3a9c1d2e"

type testCookieJar struct{ jar *cookies.CookieJar }

func (p testCookieJar) GetCookieJar() *cookies.CookieJar { return p.jar }

// cookieJarWithSession returns a jar in a temp directory holding one session
// cookie whose value is cookieTestSecret.
func cookieJarWithSession(t *testing.T) *cookies.CookieJar {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("GURL_COOKIE_DB_PATH", filepath.Join(dir, "cookies.db"))
	cdb, err := cookies.NewLMDBCookieDB()
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookies.NewCookieJar(cdb)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cdb.Close() })
	u, _ := url.Parse("https://api.example.test/")
	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: cookieTestSecret}})
	return jar
}

func TestCookiesListMasksValues(t *testing.T) {
	jar := cookieJarWithSession(t)
	newRoot := func() *cli.Command {
		root := &cli.Command{Name: "gurl", Flags: GlobalFlags(), Commands: []*cli.Command{CookiesCommand(testCookieJar{jar})}}
		root.ExitErrHandler = func(context.Context, *cli.Command, error) {}
		return root
	}

	cases := []struct {
		name       string
		args       []string
		wantReveal bool
		wantJSON   bool
		wantErr    string
	}{
		{name: "plain", args: []string{"cookies", "list"}},
		{name: "json", args: []string{"cookies", "list", "--json"}, wantJSON: true},
		{name: "agent", args: []string{"--agent", "cookies", "list"}, wantJSON: true},
		{name: "plain reveal", args: []string{"cookies", "list", "--reveal"}, wantReveal: true},
		{name: "json reveal", args: []string{"cookies", "list", "--json", "--reveal"}, wantReveal: true, wantJSON: true},
		{name: "agent reveal", args: []string{"--agent", "cookies", "list", "--reveal"}, wantErr: "--reveal is not allowed in agent mode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearAgentEnv(t)
			var err error
			stdout, stderr := captureStreams(t, func() {
				err = newRoot().Run(context.Background(), append([]string{"gurl"}, tc.args...))
			})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error %q, got %v", tc.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("%v\nstderr: %s", err, stderr)
			}
			if got := strings.Contains(stdout+stderr, cookieTestSecret); got != tc.wantReveal {
				t.Fatalf("value shown = %v, want %v:\n%s%s", got, tc.wantReveal, stdout, stderr)
			}
			if tc.wantErr != "" {
				return
			}
			if !tc.wantJSON {
				if !tc.wantReveal && !strings.Contains(stdout, redact.Mask) {
					t.Fatalf("want masked value in table:\n%s", stdout)
				}
				return
			}
			var got []cookieJSON
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("stdout is not a JSON cookie list: %v\n%s", err, stdout)
			}
			want := redact.Mask
			if tc.wantReveal {
				want = cookieTestSecret
			}
			if len(got) != 1 || got[0].Name != "session" || got[0].Value != want {
				t.Fatalf("got %+v, want one session cookie with value %q", got, want)
			}
		})
	}
}
