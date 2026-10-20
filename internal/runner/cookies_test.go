package runner

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/pkg/types"
)

// cookieServer sets a session cookie on /login and reports the session
// cookie it receives on /me through got.
func cookieServer(t *testing.T, secret string, got *string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: secret, Path: "/"})
		case "/me":
			if c, err := r.Cookie("session"); err == nil {
				*got = c.Value
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestRunnerCookieJar(t *testing.T) {
	const secret = "sess-col-91c3"
	cases := []struct {
		name string
		opts func(t *testing.T) []Option
		want string
	}{
		{
			name: "shared jar stores and sends cookies",
			opts: func(t *testing.T) []Option {
				jar, err := cookiejar.New(nil)
				if err != nil {
					t.Fatal(err)
				}
				return []Option{WithCookieJar(jar)}
			},
			want: secret,
		},
		{
			name: "no jar sends no cookies",
			opts: func(*testing.T) []Option { return nil },
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			ts := cookieServer(t, secret, &got)

			db := newMockDB()
			db.SaveRequest(&types.SavedRequest{ID: "req-login", Name: "login", URL: ts.URL + "/login", Method: "GET", Collection: "auth", SortOrder: 1})
			db.SaveRequest(&types.SavedRequest{ID: "req-me", Name: "me", URL: ts.URL + "/me", Method: "GET", Collection: "auth", SortOrder: 2})

			results, err := NewRunner(db, env.NewMemoryStore(), tc.opts(t)...).Run(context.Background(), RunConfig{CollectionName: "auth"})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if len(results) != 1 || results[0].Passed != 2 {
				t.Fatalf("expected both requests to pass, got %+v", results)
			}
			if got != tc.want {
				t.Fatalf("second request sent session cookie %q, want %q", got, tc.want)
			}
		})
	}
}
