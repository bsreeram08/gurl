package template

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// dynamicPattern matches {{$functionName}} or {{$functionName(args)}}
var dynamicPattern = regexp.MustCompile(`\{\{\$([^}]+)\}\}`)

// ResolveDynamic resolves dynamic template functions like $uuid, $timestamp, etc.
// Returns the resolved value or error for unknown functions.
func ResolveDynamic(input string) (string, error) {
	matches := dynamicPattern.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}

	result := input

	// Process from end to beginning to preserve indices when replacing
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		// Full match positions
		fullStart := match[0]
		fullEnd := match[1]
		// Capture group (function name) positions
		nameStart := match[2]
		nameEnd := match[3]

		funcNameWithArgs := input[nameStart:nameEnd]

		resolved, err := resolveFunction(funcNameWithArgs)
		if err != nil {
			return "", err
		}

		result = result[:fullStart] + resolved + result[fullEnd:]
	}

	return result, nil
}

// resolveFunction parses and executes a dynamic function by name
func resolveFunction(funcStr string) (string, error) {
	// Handle optional arguments: funcName or funcName(arg1, arg2)
	parts := strings.SplitN(funcStr, "(", 2)
	funcName := strings.TrimSpace(parts[0])
	var argsStr string
	if len(parts) > 1 {
		argsStr = strings.TrimSuffix(parts[1], ")")
	}

	switch funcName {
	case "uuid":
		if argsStr != "" {
			return "", fmt.Errorf("$uuid does not accept arguments")
		}
		return uuid.New().String(), nil

	case "timestamp":
		if argsStr != "" {
			return "", fmt.Errorf("$timestamp does not accept arguments")
		}
		return fmt.Sprintf("%d", time.Now().Unix()), nil

	case "isoTimestamp":
		if argsStr != "" {
			return "", fmt.Errorf("$isoTimestamp does not accept arguments")
		}
		return time.Now().UTC().Format(time.RFC3339), nil

	case "randomInt":
		return resolveRandomInt(argsStr)

	case "randomString":
		return resolveRandomString(argsStr)

	case "randomEmail":
		if argsStr != "" {
			return "", fmt.Errorf("$randomEmail does not accept arguments")
		}
		randomPart, err := resolveRandomString("8")
		if err != nil {
			return "", err
		}
		return randomPart + "@example.com", nil

	default:
		return "", fmt.Errorf("unknown dynamic function: $%s", funcName)
	}
}

// resolveRandomInt handles $randomInt(min, max)
func resolveRandomInt(argsStr string) (string, error) {
	if argsStr == "" {
		return "", fmt.Errorf("$randomInt requires two arguments: min, max")
	}

	// Parse min, max
	parts := strings.Split(argsStr, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("$randomInt requires two arguments: min, max")
	}

	minStr := strings.TrimSpace(parts[0])
	maxStr := strings.TrimSpace(parts[1])

	var min, max int64
	for _, c := range minStr {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("$randomInt min must be an integer")
		}
		min = min*10 + int64(c-'0')
	}
	for _, c := range maxStr {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("$randomInt max must be an integer")
		}
		max = max*10 + int64(c-'0')
	}

	if min > max {
		return "", fmt.Errorf("$randomInt min (%d) must be less than max (%d)", min, max)
	}

	// crypto/rand for cryptographic randomness
	rangeSize := max - min + 1
	n, err := rand.Int(rand.Reader, big.NewInt(rangeSize))
	if err != nil {
		return "", fmt.Errorf("failed to generate random integer: %v", err)
	}

	return fmt.Sprintf("%d", min+n.Int64()), nil
}

// resolveRandomString handles $randomString(length)
func resolveRandomString(argsStr string) (string, error) {
	if argsStr == "" {
		return "", fmt.Errorf("$randomString requires length argument")
	}

	var length int64
	for _, c := range argsStr {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("$randomString length must be an integer")
		}
		length = length*10 + int64(c-'0')
	}

	if length <= 0 {
		return "", fmt.Errorf("$randomString length must be positive")
	}

	if length > 1000 {
		return "", fmt.Errorf("$randomString length must be <= 1000")
	}

	// Alphanumeric charset
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %v", err)
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
