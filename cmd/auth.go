package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
)

func validateJSON(data []byte) error {
	var jsonTest any
	if err := json.Unmarshal(data, &jsonTest); err != nil {
		return fmt.Errorf("%w: file is not valid JSON", shared.ErrInvalidInput)
	} else {
		return nil
	}
}

func verifyAndRead(p string) ([]byte, error) {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return []byte{}, fmt.Errorf("%w: file not found: %s", shared.ErrInvalidArgument, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// AuthLogin uploads headers_auth.json to the proxy's /auth/upload endpoint.
func (r *Runner) AuthLogin(ctx context.Context, cmd *cli.Command) error {
	filePath := cmd.StringArg("path")
	fileData, err := verifyAndRead(filePath)
	authDir := filepath.Join(os.Getenv("HOME"), ".ytx")

	if err != nil {
		return err
	}

	if err := validateJSON(fileData); err != nil {
		return err
	}

	r.logger.Info("uploading auth headers", "file", filePath)

	resp, err := r.api.UploadJSON(ctx, "/auth/upload", fileData)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d, body: %s", shared.ErrAuthFailed, resp.StatusCode, string(resp.Body))
	}

	r.logger.Info("authentication successful")

	if err := os.MkdirAll(authDir, 0755); err != nil {
		r.logger.Warn("failed to create auth directory", "error", err)
	} else {
		destPath := filepath.Join(authDir, "headers_auth.json")
		if err := os.WriteFile(destPath, fileData, 0600); err != nil {
			r.logger.Warn("failed to save auth file", "error", err)
		} else {
			r.logger.Info("auth file saved", "path", destPath)
		}
	}

	return r.writePlain("✓ Authentication successful\n")
}

// AuthStatus checks current authentication state by calling the /health endpoint.
func (r *Runner) AuthStatus(ctx context.Context, cmd *cli.Command) error {
	r.logger.Info("checking auth status")

	resp, err := r.api.Get(ctx, "/health")
	if err != nil {
		return fmt.Errorf("%w: service unavailable: %v", shared.ErrServiceUnavailable, err)
	}

	if !resp.IsJSON {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return r.writePlain("✓ Service is healthy\nStatus: %s\n", string(resp.Body))
		}
		return fmt.Errorf("%w: status %d", shared.ErrServiceUnavailable, resp.StatusCode)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		healthData, ok := resp.JSONData.(map[string]any)
		if !ok {
			return r.writePlain("✓ Service is healthy\n")
		}

		status, ok := healthData["status"].(string)
		if !ok {
			status = "unknown"
		}
		authenticated := false
		if auth, ok := healthData["authenticated"].(bool); ok {
			authenticated = auth
		}

		r.writePlain("✓ Service is healthy\n")
		r.writePlain("Status: %s\n", status)
		if authenticated {
			r.writePlain("Authentication: ✓ Authenticated\n")
		} else {
			r.writePlain("Authentication: ✗ Not authenticated\n")
		}
		return nil
	}

	return fmt.Errorf("%w: status %d", shared.ErrServiceUnavailable, resp.StatusCode)
}

// authCommand handles authentication operations
func authCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Manage authentication",
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Upload headers_auth.json to FastAPI /auth/upload endpoint",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "path"},
				},
				Action: r.AuthLogin,
			},
			{
				Name:   "status",
				Usage:  "Check current authentication state (calls /health)",
				Action: r.AuthStatus,
			},
		},
	}
}
