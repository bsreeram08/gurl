package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCommandTestsRulesOffline(t *testing.T) {
	file := filepath.Join(t.TempDir(), "resp.json")
	if err := os.WriteFile(file, []byte(`{"data":{"items":[{"id":"ord_1"}],"amount":250}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		err := ExtractCommand().Run(context.Background(), []string{"extract", "-f", file, "-H", "Location: /orders/9",
			"orderId=jsonpath:$.data.items[0].id", "amt=jsonpath:$.data.amount", "loc=header:Location"})
		if err != nil {
			t.Fatalf("extract failed: %v", err)
		}
	})
	for _, want := range []string{"orderId = ord_1", "amt = 250", "loc = /orders/9"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in output:\n%s", want, output)
		}
	}

	var err error
	output = captureStdout(t, func() {
		err = ExtractCommand().Run(context.Background(), []string{"extract", "-f", file, "x=jsonpath:$.nope"})
	})
	if err == nil || !strings.Contains(output, "x = (no match)") {
		t.Fatalf("expected no-match failure, got err=%v output=%s", err, output)
	}
}
