package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// InteractiveConfig holds configuration for interactive mode
type InteractiveConfig struct {
	PrettyPrint bool
	Color       bool
	ShowTime    bool
}

// InteractiveRunner manages the interactive WebSocket session
type InteractiveRunner struct {
	client *Client
	stdin  *os.File
	stdout *os.File
	stderr *os.File
	config InteractiveConfig
}

// NewInteractiveRunner creates a new interactive runner
func NewInteractiveRunner(client *Client, cfg InteractiveConfig) *InteractiveRunner {
	return &InteractiveRunner{
		client: client,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		config: cfg,
	}
}

// Run starts the interactive WebSocket session
// This method blocks until the session ends
func (r *InteractiveRunner) Run(ctx context.Context) error {
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start receive goroutine
	errChan := make(chan error, 1)
	msgChan := make(chan string, 1)

	// Start the receive loop in a goroutine
	go r.receiveLoop(ctx, msgChan, errChan)

	// Start stdin reader in a goroutine
	go r.stdinLoop(ctx, cancel, msgChan, errChan)

	// Main loop - wait for signals or errors
	for {
		select {
		case <-sigChan:
			// Received interrupt signal
			fmt.Fprintf(r.stderr, "\nReceived interrupt, closing connection...\n")
			cancel()
			r.client.Close()
			return nil

		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(r.stderr, "Error: %v\n", err)
				return err
			}

		case msg := <-msgChan:
			fmt.Fprintln(r.stdout, msg)
		}
	}
}

// receiveLoop continuously receives messages and sends them to msgChan
func (r *InteractiveRunner) receiveLoop(ctx context.Context, msgChan chan<- string, errChan chan<- error) {
	defer close(msgChan)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgType, data, err := r.client.NextReader()
			if err != nil {
				select {
				case errChan <- fmt.Errorf("receive error: %w", err):
				case <-ctx.Done():
				}
				return
			}

			// Handle different message types
			switch MessageType(msgType) {
			case MessageTypeText:
				msg := r.formatMessage(data, "RECV")
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}

			case MessageTypeBinary:
				msg := r.formatBinaryMessage(data, "RECV")
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}

			case MessageTypeClose:
				select {
				case msgChan <- "Connection closed by server":
				case <-ctx.Done():
				}
				return

			case MessageTypePing:
				// Respond with pong - must include the payload from the ping
				if err := r.client.Send(data, MessageTypePong); err != nil {
					select {
					case errChan <- fmt.Errorf("pong failed: %w", err):
					case <-ctx.Done():
					}
					return
				}

			case MessageTypePong:
				// Pong received, nothing to do
			}
		}
	}
}

// stdinLoop reads from stdin and sends to WebSocket
func (r *InteractiveRunner) stdinLoop(ctx context.Context, cancel context.CancelFunc, msgChan chan<- string, errChan chan<- error) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Set a deadline for read to allow checking context periodically
			r.stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := r.stdin.Read(buf)
			if err != nil {
				if os.IsTimeout(err) {
					continue // Check context and retry
				}
				// Stdin closed or error
				select {
				case errChan <- nil: // Signal end of input
				case <-ctx.Done():
				}
				cancel()
				return
			}

			if n > 0 {
				input := string(buf[:n])
				// Remove trailing newline
				input = trimNewline(input)

				// Try to parse as JSON for pretty printing
				var jsonData interface{}
				if err := json.Unmarshal([]byte(input), &jsonData); err == nil {
					// Valid JSON, send formatted
					prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
					if err := r.client.SendText(string(prettyJSON)); err != nil {
						select {
						case errChan <- fmt.Errorf("send error: %w", err):
						case <-ctx.Done():
						}
						return
					}
				} else {
					// Not JSON, send as-is
					if err := r.client.SendText(input); err != nil {
						select {
						case errChan <- fmt.Errorf("send error: %w", err):
						case <-ctx.Done():
						}
						return
					}
				}

				msg := r.formatMessage([]byte(input), "SENT")
				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// formatMessage formats a text message for display
func (r *InteractiveRunner) formatMessage(data []byte, prefix string) string {
	if r.config.ShowTime {
		timestamp := time.Now().Format("15:04:05.000")
		if r.config.PrettyPrint {
			var jsonData interface{}
			if err := json.Unmarshal(data, &jsonData); err == nil {
				prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
				return fmt.Sprintf("[%s] %s:\n%s", timestamp, prefix, string(prettyJSON))
			}
		}
		return fmt.Sprintf("[%s] %s: %s", timestamp, prefix, string(data))
	}

	if r.config.PrettyPrint {
		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			return fmt.Sprintf("%s:\n%s", prefix, string(prettyJSON))
		}
	}
	return fmt.Sprintf("%s: %s", prefix, string(data))
}

// formatBinaryMessage formats binary data for display
func (r *InteractiveRunner) formatBinaryMessage(data []byte, prefix string) string {
	timestamp := ""
	if r.config.ShowTime {
		timestamp = time.Now().Format("15:04:05.000")
	}

	// Show as hex dump for binary
	hexStr := formatHex(data, 16)
	if timestamp != "" {
		return fmt.Sprintf("[%s] %s (binary %d bytes):\n%s", timestamp, prefix, len(data), hexStr)
	}
	return fmt.Sprintf("%s (binary %d bytes):\n%s", prefix, len(data), hexStr)
}

// formatHex creates a hex dump string
func formatHex(data []byte, bytesPerLine int) string {
	var result strings.Builder
	result.Grow(len(data) * 4) // Pre-allocate

	for i := 0; i < len(data); i += bytesPerLine {
		end := i + bytesPerLine
		if end > len(data) {
			end = len(data)
		}
		line := data[i:end]

		// Build hex and ascii parts
		var hexBuilder, asciiBuilder strings.Builder
		hexBuilder.Grow(bytesPerLine * 3)
		asciiBuilder.Grow(bytesPerLine)

		for _, b := range line {
			hexBuilder.WriteString(fmt.Sprintf("%02x ", b))
			if b >= 32 && b <= 126 {
				asciiBuilder.WriteByte(b)
			} else {
				asciiBuilder.WriteByte('.')
			}
		}
		hex := hexBuilder.String()
		ascii := asciiBuilder.String()

		// Pad hex to full width
		for len(line) < bytesPerLine {
			hex += "   "
		}
		result.WriteString(fmt.Sprintf("  %04x: %s|%s|\n", i, hex, ascii))
	}
	return result.String()
}

// trimNewline removes trailing newline characters
func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// SendAndReceive runs a one-shot send and receive
func (r *InteractiveRunner) SendAndReceive(ctx context.Context, data string) (string, error) {
	// Send the data
	if err := r.client.SendText(data); err != nil {
		return "", fmt.Errorf("send failed: %w", err)
	}

	// Set a timeout for receiving response
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Receive response
	respData, msgType, err := r.client.Receive()
	if err != nil {
		return "", fmt.Errorf("receive failed: %w", err)
	}

	if msgType == MessageTypeText {
		return string(respData), nil
	}

	return string(respData), fmt.Errorf("unexpected message type: %v", msgType)
}
