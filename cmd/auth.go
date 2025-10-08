package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/desertthunder/ytx/internal/shared"
	"github.com/urfave/cli/v3"
)

// AuthLogin uploads headers_auth.json to the proxy's /auth/upload endpoint.
func (r *Runner) AuthLogin(ctx context.Context, cmd *cli.Command) error {
	filePath := cmd.StringArg("path")
	fileData, err := shared.VerifyAndReadFile(filePath)
	authDir := filepath.Join(os.Getenv("HOME"), ".ytx")

	if err != nil {
		return err
	}

	if err := shared.ValidateJSON(fileData); err != nil {
		return err
	}

	r.logger.Infof("uploading auth headers to %v", filePath)

	resp, err := r.api.UploadJSON(ctx, "/auth/upload", fileData)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d, body: %s", shared.ErrAuthFailed, resp.StatusCode, string(resp.Body))
	}

	r.logger.Info("authentication successful")

	if err := os.MkdirAll(authDir, 0755); err != nil {
		r.logger.Warnf("failed to create auth director %v", err)
	} else {
		destPath := filepath.Join(authDir, "headers_auth.json")
		if err := os.WriteFile(destPath, fileData, 0600); err != nil {
			r.logger.Warnf("failed to save auth file %v", err)
		} else {
			r.logger.Infof("auth file saved to %v", destPath)
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
