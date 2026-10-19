package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sreeram/gurl/internal/extract"
	"github.com/sreeram/gurl/pkg/types"
	"github.com/urfave/cli/v3"
)

const extractRuleUsage = "Extraction rule VAR=METHOD:EXPR. METHOD: jsonpath ($.data.id, $.items[0].id), header (Location), regex (first capture group), jq (alias of jsonpath). Try rules offline with 'gurl extract'"

// ExtractCommand tests extraction rules against a sample response without sending a request.
func ExtractCommand() *cli.Command {
	return &cli.Command{
		Name:      "extract",
		Usage:     "Test extraction rules against a sample response (no request is sent)",
		ArgsUsage: "VAR=METHOD:EXPR [VAR=METHOD:EXPR...]",
		Description: `Reads a response body from --file or stdin and prints what each rule extracts.

Examples:
  echo '{"data":{"id":"ord_1"}}' | gurl extract 'orderId=jsonpath:$.data.id'
  gurl extract -f resp.json -H 'Location: /orders/9' 'loc=header:Location' 'n=regex:"amount":(\d+)'`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "Response body file (default: stdin)",
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "Response header 'Key: Value' for header rules (can repeat)",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.Args().Len() == 0 {
				return fmt.Errorf("at least one rule is required (VAR=METHOD:EXPR)")
			}
			rules, err := parseExtractFlags(c.Args().Slice())
			if err != nil {
				return err
			}

			var body []byte
			if path := c.String("file"); path != "" {
				body, err = os.ReadFile(path)
			} else {
				body, err = io.ReadAll(os.Stdin)
			}
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			headers := make(map[string][]string)
			for _, h := range c.StringSlice("header") {
				key, value, ok := strings.Cut(h, ":")
				if !ok {
					return fmt.Errorf("invalid header %q: must be 'Key: Value'", h)
				}
				key = strings.TrimSpace(key)
				headers[key] = append(headers[key], strings.TrimSpace(value))
			}

			extractor := extract.NewExtractor()
			missing := 0
			for _, rule := range rules {
				values, err := extractor.Extract(body, headers, []types.Extract{rule})
				if err != nil {
					return fmt.Errorf("%s: %w", rule.Name, err)
				}
				if values[rule.Name] == "" {
					missing++
					fmt.Printf("%s = (no match)\n", rule.Name)
					continue
				}
				fmt.Printf("%s = %s\n", rule.Name, values[rule.Name])
			}
			if missing > 0 {
				return fmt.Errorf("%d rule(s) matched nothing", missing)
			}
			return nil
		},
	}
}
