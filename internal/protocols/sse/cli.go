package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

func SSECommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "sse",
		Aliases: []string{"events", "sse"},
		Usage:   "Connect to a Server-Sent Events (SSE) endpoint",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "Custom headers (key:value format, use multiple times for multiple headers)",
			},
			&cli.StringFlag{
				Name:    "event",
				Aliases: []string{"e"},
				Usage:   "Filter by event type (can be specified multiple times)",
			},
			&cli.StringFlag{
				Name:    "last-event-id",
				Aliases: []string{"L"},
				Usage:   "Last-Event-ID header value for reconnect",
			},
			&cli.IntFlag{
				Name:    "timeout",
				Aliases: []string{"T"},
				Usage:   "Connection timeout in seconds",
				Value:   30,
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Parse and pretty-print JSON data",
			},
			&cli.BoolFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Enable syntax highlighting for JSON output",
			},
			&cli.BoolFlag{
				Name:    "timestamp",
				Aliases: []string{"t"},
				Usage:   "Show timestamps for each event",
			},
			&cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "Output raw data without formatting",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			url := c.Args().Get(0)
			if url == "" {
				return fmt.Errorf("SSE endpoint URL is required")
			}

			opts := []Option{}

			if headerStr := c.String("header"); headerStr != "" {
				headerPairs := strings.Split(headerStr, ",")
				for _, pair := range headerPairs {
					parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
					if len(parts) == 2 {
						opts = append(opts, WithHeader(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])))
					}
				}
			}

			if eventType := c.String("event"); eventType != "" {
				opts = append(opts, WithEventType(eventType))
			}

			if lastEventID := c.String("last-event-id"); lastEventID != "" {
				opts = append(opts, WithLastEventID(lastEventID))
			}

			timeout := time.Duration(c.Int("timeout")) * time.Second
			if timeout > 0 {
				opts = append(opts, WithTimeout(timeout))
			}

			client := NewClient()

			eventChan, errorChan, err := client.Connect(ctx, url, opts...)
			if err != nil {
				return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
			}

			showTimestamp := c.Bool("timestamp")
			prettyJSON := c.Bool("json")
			_ = c.Bool("color") // reserved for future color formatting
			raw := c.Bool("raw")

			for {
				select {
				case event, ok := <-eventChan:
					if !ok {
						return nil
					}

					if raw {
						fmt.Println(event.Data)
					} else if prettyJSON && event.Data != "" {
						if showTimestamp {
							fmt.Printf("[%s] ", time.Now().Format(time.RFC3339))
						}
						if event.Type != "" {
							fmt.Printf("[%s] ", event.Type)
						}

						var jsonData interface{}
						if err := json.Unmarshal([]byte(event.Data), &jsonData); err == nil {
							output, _ := json.MarshalIndent(jsonData, "", "  ")
							fmt.Println(string(output))
						} else {
							fmt.Println(event.Data)
						}
					} else {
						if showTimestamp {
							fmt.Printf("[%s] ", time.Now().Format(time.RFC3339))
						}
						if event.Type != "" {
							fmt.Printf("[%s] ", event.Type)
						}
						if event.ID != "" {
							fmt.Printf("(id=%s) ", event.ID)
						}
						if event.Retry > 0 {
							fmt.Printf("(retry=%dms) ", event.Retry)
						}
						fmt.Println(event.Data)
					}

				case err, ok := <-errorChan:
					if !ok {
						return nil
					}
					return fmt.Errorf("SSE error: %w", err)

				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
	}
}
