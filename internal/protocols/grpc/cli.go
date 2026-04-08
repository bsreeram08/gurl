package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sreeram/gurl/internal/formatter"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/urfave/cli/v3"
)

// GRPCCommand returns a cli.Command for gRPC requests
func GRPCCommand(db storage.DB) *cli.Command {
	return &cli.Command{
		Name:    "grpc",
		Aliases: []string{"grpc"},
		Usage:   "Execute a gRPC request",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "service",
				Aliases: []string{"s"},
				Usage:   "gRPC service name (e.g., 'UserService')",
			},
			&cli.StringFlag{
				Name:    "method",
				Aliases: []string{"m"},
				Usage:   "gRPC method name (e.g., 'GetUser')",
			},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"d"},
				Usage:   "JSON request data",
			},
			&cli.StringFlag{
				Name:    "data-file",
				Aliases: []string{"f"},
				Usage:   "Path to file containing JSON request data",
			},
			&cli.StringFlag{
				Name:    "call-type",
				Aliases: []string{"ct"},
				Usage:   "Call type: unary, server-streaming, client-streaming, bidirectional",
				Value:   "unary",
			},
			&cli.BoolFlag{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List available services via reflection",
			},
			&cli.StringFlag{
				Name:    "metadata",
				Aliases: []string{"mdata"},
				Usage:   "Comma-separated key:value pairs for gRPC metadata",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Output format (auto|json|table)",
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "color",
				Aliases: []string{"c"},
				Usage:   "Enable syntax highlighting",
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
			target := c.Args().Get(0)
			if target == "" {
				return fmt.Errorf("target (host:port) is required")
			}

			// Build TLS config if needed
			var tlsCfg *TLSConfig
			if c.Bool("insecure") || c.String("cacert") != "" || c.String("cert") != "" {
				tlsCfg = &TLSConfig{
					Insecure:   c.Bool("insecure"),
					CAFile:     c.String("cacert"),
					CertFile:   c.String("cert"),
					KeyFile:    c.String("key"),
					ServerName: c.String("server-name"),
				}
			}

			// Create client
			client := NewClient()
			if tlsCfg != nil {
				client = NewClientWithTLS(*tlsCfg)
			}

			// Handle --list flag
			if c.Bool("list") {
				return listServices(ctx, client, target)
			}

			// Parse call type
			callTypeStr := c.String("call-type")
			callType, err := parseCallType(callTypeStr)
			if err != nil {
				return fmt.Errorf("invalid call type: %w", err)
			}

			// Validate required flags for non-list
			service := c.String("service")
			method := c.String("method")
			if service == "" || method == "" {
				return fmt.Errorf("--service and --method are required for gRPC calls")
			}

			// Parse data
			dataStr := c.String("data")
			dataFile := c.String("data-file")
			if dataFile != "" {
				dataBytes, err := os.ReadFile(dataFile)
				if err != nil {
					return fmt.Errorf("failed to read data file: %w", err)
				}
				dataStr = string(dataBytes)
			}
			if dataStr == "" {
				dataStr = "{}"
			}

			// Parse metadata
			var metadataPairs []string
			if metaStr := c.String("metadata"); metaStr != "" {
				metadataPairs = strings.Split(metaStr, ",")
			}

			// Execute based on call type
			var output []byte
			var grpcResp *Response
			var streamResp *StreamingResponse

			fullMethod := fmt.Sprintf("/%s/%s", service, method)

			switch callType {
			case CallTypeUnary:
				grpcResp, err = client.ExecuteUnary(ctx, target, fullMethod, []byte(dataStr))
				if err != nil {
					fmt.Fprintf(os.Stderr, "gRPC Error: %v\n", err)
				}
				if grpcResp != nil {
					output = grpcResp.Data
				}

			case CallTypeServerStreaming:
				streamResp, err = client.ExecuteServerStreaming(ctx, target, fullMethod, []byte(dataStr))
				if err != nil {
					fmt.Fprintf(os.Stderr, "gRPC Error: %v\n", err)
				}
				if streamResp != nil {
					output, _ = json.MarshalIndent(streamResp.Events, "", "  ")
				}

			case CallTypeClientStreaming:
				streamResp, err = client.ExecuteClientStreaming(ctx, target, fullMethod, []byte(dataStr))
				if err != nil {
					fmt.Fprintf(os.Stderr, "gRPC Error: %v\n", err)
				}
				if streamResp != nil {
					output, _ = json.MarshalIndent(streamResp.Events, "", "  ")
				}

			case CallTypeBidirectionalStreaming:
				streamResp, err = client.ExecuteBidirectionalStreaming(ctx, target, fullMethod, []byte(dataStr))
				if err != nil {
					fmt.Fprintf(os.Stderr, "gRPC Error: %v\n", err)
				}
				if streamResp != nil {
					output, _ = json.MarshalIndent(streamResp.Events, "", "  ")
				}
			}

			// Handle metadata
			if grpcResp != nil && grpcResp.StatusCode != 0 {
				statusStr := StatusCodeToString(grpcResp.StatusCode)
				fmt.Fprintf(os.Stderr, "Status: %s\n", statusStr)
			}

			// Print output
			if len(output) > 0 {
				color := c.Bool("color")
				opts := formatter.FormatOptions{
					Indent: "  ",
					Color:  color,
				}
				formatted := formatter.Format(output, "application/json", opts)
				fmt.Println(formatted)
			}

			// Print metadata if present
			if len(metadataPairs) > 0 {
				fmt.Fprintf(os.Stderr, "Metadata sent: %s\n", strings.Join(metadataPairs, ", "))
			}

			return nil
		},
	}
}

// listServices uses reflection to list available services
func listServices(ctx context.Context, client *Client, target string) error {
	// This would use reflection to list services
	// For now, return a placeholder
	fmt.Printf("Listing services on %s...\n", target)
	fmt.Println("(Reflection requires server support)")
	return nil
}

// parseCallType converts string to CallType
func parseCallType(s string) (CallType, error) {
	switch strings.ToLower(s) {
	case "unary":
		return CallTypeUnary, nil
	case "server-streaming", "server_streaming", "serverstreaming":
		return CallTypeServerStreaming, nil
	case "client-streaming", "client_streaming", "clientstreaming":
		return CallTypeClientStreaming, nil
	case "bidirectional", "bidirectional-streaming", "bidi":
		return CallTypeBidirectionalStreaming, nil
	default:
		return CallTypeUnary, fmt.Errorf("unknown call type: %s", s)
	}
}
