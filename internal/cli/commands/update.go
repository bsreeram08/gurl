package commands

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"
)

// CurrentVersion is set at build time via ldflags
var CurrentVersion = "dev"

// UpdateCommand checks for and applies updates
func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update gurl to the latest version",
		Action: func(ctx context.Context, c *cli.Command) error {
			return updateGurl()
		},
	}
}

const (
	owner       = "bsreeram08"
	repo        = "gurl"
	latestURL   = "https://api.github.com/repos/%s/%s/releases/latest"
	downloadURL = "https://github.com/%s/%s/releases/download/v%s/gurl-%s-%s"
)

func updateGurl() error {
	currentVersion := CurrentVersion
	fmt.Printf("Current version: %s\n", currentVersion)

	// Get latest release
	resp, err := http.Get(fmt.Sprintf(latestURL, owner, repo))
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	// Parse response to get latest version tag
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	bodyStr := string(body)

	// Extract tag_name
	tagStart := strings.Index(bodyStr, `"tag_name":"`)
	if tagStart == -1 {
		return fmt.Errorf("failed to parse latest release")
	}
	tagStart += 12
	tagEnd := strings.Index(bodyStr[tagStart:], `"`)
	if tagEnd == -1 {
		return fmt.Errorf("failed to parse tag name")
	}
	latestVersion := bodyStr[tagStart : tagStart+tagEnd]
	latestVersion = strings.TrimPrefix(latestVersion, "v")

	fmt.Printf("Latest version: %s\n", latestVersion)

	// Compare versions (strip v prefix and -dirty suffix for consistent comparison)
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	currentVersion = strings.SplitN(currentVersion, "-dirty", 2)[0]
	if latestVersion == currentVersion || currentVersion == "dev" || strings.HasPrefix(currentVersion, "dev-") {
		fmt.Println("Already up to date!")
		return nil
	}

	// Determine OS and arch
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize names
	switch osName {
	case "darwin":
		osName = "darwin"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Build download URL
	downloadLink := fmt.Sprintf(downloadURL, owner, repo, latestVersion, osName, arch)

	// Get checksum file URL
	checksumURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/SHA256SUMS", owner, repo, latestVersion)

	fmt.Printf("Downloading from: %s\n", downloadLink)

	// Download new binary
	resp, err = http.Get(downloadLink)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "gurl-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpBin := tmpFile.Name()
	defer os.Remove(tmpBin)

	// Copy to temp
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	os.Chmod(tmpBin, 0755)

	// Verify checksum
	fmt.Println("Verifying checksum...")
	checksumResp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer checksumResp.Body.Close()

	if checksumResp.StatusCode != 200 {
		return fmt.Errorf("failed to download checksum file: HTTP %d", checksumResp.StatusCode)
	}

	checksumBody, err := io.ReadAll(checksumResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum file: %w", err)
	}

	// Parse SHA256SUMS — format: "<hash>  <binary-name>" per line
	// e.g., "abc123def...  gurl-darwin-arm64"
	expectedHash := ""
	binaryName := fmt.Sprintf("gurl-%s-%s", osName, arch)
	for _, line := range strings.Split(string(checksumBody), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <name>" (two spaces between hash and name)
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		hash := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if name == binaryName {
			expectedHash = hash
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum for %s not found in SHA256SUMS", binaryName)
	}

	// Compute SHA256 of downloaded binary
	tmpFile, err = os.Open(tmpBin)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer tmpFile.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, tmpFile); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	computedHash := hex.EncodeToString(hash.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(computedHash), []byte(expectedHash)) != 1 {
		os.Remove(tmpBin)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, computedHash)
	}

	// Replace current binary
	selfPath, err := os.Executable()
	if err != nil {
		// Fallback: try to find gurl in PATH
		path, err := exec.LookPath("gurl")
		if err != nil {
			return fmt.Errorf("could not find gurl executable: %w", err)
		}
		selfPath = path
	}

	fmt.Printf("Replacing: %s\n", selfPath)

	// On Unix, we need to remove the old binary first
	// On Windows, we might need to close handles first
	backupPath := selfPath + ".old"
	os.Rename(selfPath, backupPath)
	os.Rename(tmpBin, selfPath)
	os.Chmod(selfPath, 0755)

	// Try to remove backup (may fail on Windows if file is in use)
	os.Remove(backupPath)

	fmt.Printf("Successfully updated to v%s!\n", latestVersion)
	fmt.Println("Run 'gurl --version' to verify.")

	return nil
}
