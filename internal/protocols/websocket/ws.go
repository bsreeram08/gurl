package websocket

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents WebSocket message types
type MessageType int

const (
	MessageTypeText   MessageType = websocket.TextMessage
	MessageTypeBinary MessageType = websocket.BinaryMessage
	MessageTypeClose  MessageType = websocket.CloseMessage
	MessageTypePing   MessageType = websocket.PingMessage
	MessageTypePong   MessageType = websocket.PongMessage
)

// ReconnectConfig holds reconnection settings
type ReconnectConfig struct {
	Enabled    bool
	MaxRetries int
	Backoff    time.Duration
}

// Client wraps a WebSocket connection with gorilla/websocket
type Client struct {
	conn           *websocket.Conn
	url            string
	headers        http.Header
	reconnect      ReconnectConfig
	tlsConfig      *tls.Config
	mu             sync.RWMutex
	ioMu           sync.Mutex
	closed         bool
	messageHandler func([]byte, MessageType) // Optional handler for received messages
}

// NewClient creates a new WebSocket client
func NewClient() *Client {
	return &Client{
		reconnect: ReconnectConfig{
			Enabled:    false,
			MaxRetries: 3,
			Backoff:    time.Second,
		},
	}
}

// SetReconnect configures reconnection behavior
func (c *Client) SetReconnect(enabled bool, maxRetries int, backoff time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reconnect = ReconnectConfig{
		Enabled:    enabled,
		MaxRetries: maxRetries,
		Backoff:    backoff,
	}
}

// SetHeaders sets custom headers to be sent during WebSocket handshake
func (c *Client) SetHeaders(headers http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers = headers
}

// SetMessageHandler sets a callback for received messages
func (c *Client) SetMessageHandler(handler func([]byte, MessageType)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageHandler = handler
}

// SetTLSConfig sets TLS configuration for the WebSocket connection
func (c *Client) SetTLSConfig(cfg *tls.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tlsConfig = cfg
}

// Connect establishes a WebSocket connection to the specified URL
func (c *Client) Connect(ctx context.Context, url string, headers http.Header) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	c.url = url
	c.headers = headers.Clone()
	c.mu.Unlock()

	return c.connectWithRetry(ctx, 0)
}

// connectWithRetry attempts to connect with reconnection support
func (c *Client) connectWithRetry(ctx context.Context, attempt int) error {
	c.mu.RLock()
	tlsConfig := c.tlsConfig
	c.mu.RUnlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Apply TLS config if set
	if tlsConfig != nil {
		dialer.TLSClientConfig = tlsConfig
	}

	conn, _, err := dialer.DialContext(ctx, c.url, c.headers)
	if err != nil {
		// Check if we should retry
		c.mu.RLock()
		retryEnabled := c.reconnect.Enabled
		maxRetries := c.reconnect.MaxRetries
		backoff := c.reconnect.Backoff
		c.mu.RUnlock()

		if retryEnabled && attempt < maxRetries {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				return c.connectWithRetry(ctx, attempt+1)
			}
		}
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	return nil
}

// Send sends a text message to the WebSocket connection
func (c *Client) SendText(msg string) error {
	return c.Send([]byte(msg), MessageTypeText)
}

// SendJSON sends a JSON message to the WebSocket connection
func (c *Client) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Send(data, MessageTypeText)
}

// Send sends a message with the specified type
func (c *Client) Send(msg []byte, msgType MessageType) error {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mu.RUnlock()

	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	return conn.WriteMessage(int(msgType), msg)
}

// Receive receives a message from the WebSocket connection
// Returns the message data, message type, and any error
func (c *Client) Receive() ([]byte, MessageType, error) {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return nil, 0, fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mu.RUnlock()

	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read message: %w", err)
	}

	return data, MessageType(msgType), nil
}

// ReceiveMultiple starts a goroutine to continuously receive messages
// and send them to the returned channel. The goroutine exits when the context
// is cancelled or the connection is closed.
func (c *Client) ReceiveMultiple(ctx context.Context) (<-chan Message, error) {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	msgChan := make(chan Message, 100) // Buffered channel

	go func() {
		defer close(msgChan)
		for {
			// Get current connection reference fresh each iteration
			// This ensures we use the latest conn after any reconnect
			c.mu.RLock()
			currentConn := c.conn
			c.mu.RUnlock()

			select {
			case <-ctx.Done():
				return
			default:
				c.ioMu.Lock()
				currentConn.SetReadDeadline(time.Now().Add(60 * time.Second))
				msgType, data, err := currentConn.ReadMessage()
				c.ioMu.Unlock()
				if err != nil {
					// Check if it's a close error
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						// Try to reconnect if enabled
						c.mu.RLock()
						retryEnabled := c.reconnect.Enabled
						c.mu.RUnlock()

						if retryEnabled {
							if reconnectErr := c.connectWithRetry(ctx, 0); reconnectErr == nil {
								// Get fresh conn reference after successful reconnect
								c.mu.RLock()
								currentConn = c.conn
								c.mu.RUnlock()
								continue
							}
						}
					}
					return
				}

				msg := Message{
					Type: MessageType(msgType),
					Data: data,
				}

				select {
				case msgChan <- msg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return msgChan, nil
}

// Message represents a received WebSocket message
type Message struct {
	Type MessageType
	Data []byte
}

// Ping sends a ping message to the WebSocket connection
func (c *Client) Ping() error {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mu.RUnlock()

	return conn.WriteMessage(websocket.PingMessage, nil)
}

// Close gracefully closes the WebSocket connection
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}

	if c.conn == nil {
		c.closed = true
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	// Write close frame and wait for server response
	c.ioMu.Lock()
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		c.ioMu.Unlock()
		// Try to close anyway
		c.mu.Lock()
		c.conn.Close()
		c.closed = true
		c.mu.Unlock()
		return fmt.Errorf("failed to send close frame: %w", err)
	}

	// Wait for close frame from server - set a deadline
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	// Read the close response from server (may be a close message or error)
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// If it's a close error, that's expected
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			// For other errors (including EOF on close), break
			break
		}
	}

	c.ioMu.Unlock()

	c.mu.Lock()
	err = c.conn.Close()
	c.closed = true
	c.mu.Unlock()
	return err
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.closed
}

// Reconnect attempts to reconnect to the WebSocket server
func (c *Client) Reconnect(ctx context.Context) error {
	// Close existing connection if any
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	// Reconnect is allowed after Close() — reset closed flag
	c.closed = false
	c.mu.Unlock()

	return c.connectWithRetry(ctx, 0)
}

// SendBinary sends binary data through the WebSocket
func (c *Client) SendBinary(data []byte) error {
	return c.Send(data, MessageTypeBinary)
}

// NextReader returns the next message reader from the connection
// This allows reading messages in a streaming fashion
func (c *Client) NextReader() (MessageType, []byte, error) {
	c.mu.RLock()
	if c.conn == nil {
		c.mu.RUnlock()
		return 0, nil, fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mu.RUnlock()

	msgType, data, err := conn.ReadMessage()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read message: %w", err)
	}

	return MessageType(msgType), data, nil
}
