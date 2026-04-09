package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

// WSCommand returns a cli.Command for WebSocket requests
func WSCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "ws",
		Aliases: []string{"websocket"},
		Usage:   "Execute a WebSocket request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "Custom headers (key:value format)",
			},
			&cli.StringFlag{
				Name:    "send",
				Aliases: []string{"s"},
				Usage:   "Send a message and exit (one-shot mode)",
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Interactive mode (stdin/stdout) - default if no --send",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "pretty",
				Aliases: []string{"p"},
				Usage:   "Pretty print JSON messages",
			},
			&cli.BoolFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Enable syntax highlighting",
			},
			&cli.BoolFlag{
				Name:    "timestamp",
				Aliases: []string{"t"},
				Usage:   "Show timestamps in output",
			},
			&cli.IntFlag{
				Name:    "timeout",
				Aliases: []string{"T"},
				Usage:   "Connection timeout in seconds",
				Value:   30,
			},
			&cli.BoolFlag{
				Name:    "reconnect",
				Aliases: []string{"r"},
				Usage:   "Enable auto-reconnect on disconnect",
			},
			&cli.IntFlag{
				Name:  "max-retries",
				Usage: "Maximum reconnection attempts",
				Value: 3,
			},
			// TLS flags
			&cli.BoolFlag{
				Name:    "insecure",
				Aliases: []string{"k"},
				Usage:   "Skip TLS verification (use for testing only)",
			},
			&cli.StringFlag{
				Name:  "cacert",
				Usage: "CA certificate file",
			},
			&cli.StringFlag{
				Name:  "cert",
				Usage: "Client certificate file",
			},
			&cli.StringFlag{
				Name:  "key",
				Usage: "Client key file",
			},
			&cli.StringFlag{
				Name:  "server-name",
				Usage: "Server name for SNI",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			url := c.Args().Get(0)
			if url == "" {
				return fmt.Errorf("WebSocket URL is required")
			}

			// Parse headers
			headers := make(http.Header)
			if headerStr := c.String("header"); headerStr != "" {
				headerPairs := strings.Split(headerStr, ",")
				for _, pair := range headerPairs {
					parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
					if len(parts) == 2 {
						headers.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
					}
				}
			}

			// Create client
			client := NewClient()

			// Configure reconnect if enabled
			if c.Bool("reconnect") {
				client.SetReconnect(true, c.Int("max-retries"), time.Second)
			}

			// Connect with timeout
			connectCtx, cancel := context.WithTimeout(ctx, time.Duration(c.Int("timeout"))*time.Second)
			defer cancel()

			if err := client.Connect(connectCtx, url, headers); err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			defer client.Close()

			// Handle --send (one-shot mode)
			if sendData := c.String("send"); sendData != "" {
				return handleOneShot(client, sendData, c.Bool("pretty"), c.Bool("color"))
			}

			// Interactive mode
			cfg := InteractiveConfig{
				PrettyPrint: c.Bool("pretty"),
				Color:       c.Bool("color"),
				ShowTime:    c.Bool("timestamp"),
			}
			runner := NewInteractiveRunner(client, cfg)
			return runner.Run(ctx)
		},
	}
}

// handleOneShot sends a single message and displays the response
func handleOneShot(client *Client, data string, pretty, color bool) error {
	// Try to parse as JSON for sending
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
		// Valid JSON, send with formatting
		if err := client.SendJSON(jsonData); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
	} else {
		// Not JSON, send as text
		if err := client.SendText(data); err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
	}

	// Receive response with timeout
	respData, msgType, err := client.Receive()
	if err != nil {
		return fmt.Errorf("receive failed: %w", err)
	}

	if msgType == MessageTypeText {
		output := respData
		if pretty {
			var jsonData interface{}
			if err := json.Unmarshal(respData, &jsonData); err == nil {
				formatted, _ := json.MarshalIndent(jsonData, "", "  ")
				output = formatted
			}
		}
		fmt.Println(string(output))
	} else if msgType == MessageTypeBinary {
		// Show binary as hex dump
		fmt.Printf("Binary message (%d bytes):\n%s\n", len(respData), formatHexDump(respData))
	}

	return nil
}

// formatHexDump creates a hex dump for binary data
func formatHexDump(data []byte) string {
	const bytesPerLine = 16
	result := ""
	for i := 0; i < len(data); i += bytesPerLine {
		end := i + bytesPerLine
		if end > len(data) {
			end = len(data)
		}
		line := data[i:end]
		hex := ""
		ascii := ""
		for _, b := range line {
			hex += fmt.Sprintf("%02x ", b)
			if b >= 32 && b <= 126 {
				ascii += string(b)
			} else {
				ascii += "."
			}
		}
		for len(line) < bytesPerLine {
			hex += "   "
		}
		result += fmt.Sprintf("  %04x: %s|%s|\n", i, hex, ascii)
	}
	return result
}

// DialerConfig holds configuration for WebSocket dialer
type DialerConfig struct {
	Header     http.Header
	Timeout    time.Duration
	Insecure   bool
	CAFile     string
	CertFile   string
	KeyFile    string
	ServerName string
}

// NewDialerWithTLS creates a websocket.Dialer with TLS configuration
func NewDialerWithTLS(cfg DialerConfig) (*websocket.Dialer, error) {
	dialer := &websocket.Dialer{
		HandshakeTimeout: cfg.Timeout,
	}

	if cfg.Insecure || cfg.CertFile != "" || cfg.ServerName != "" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
			ServerName:         cfg.ServerName,
		}
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		dialer.TLSClientConfig = tlsConfig
	}

	return dialer, nil
}
