package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type HookCommand struct {
	Type    string `json:"type,omitempty"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type HookMatcher struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []HookCommand `json:"hooks"`
}

type HooksConfig struct {
	Hooks map[string][]HookMatcher `json:"hooks"`
}

type InstalledResponse struct {
	Installed bool `json:"installed"`
}

type InstallHooksResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

func getAntigravityDir(targetDir string) string {
	if targetDir == "" {
		targetDir = "."
	}
	return filepath.Join(targetDir, ".antigravity")
}

func handleInstallHooks(targetDir string) error {
	dirPath := getAntigravityDir(targetDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	hooksFile := filepath.Join(dirPath, "hooks.json")
	hookConfig := HooksConfig{
		Hooks: map[string][]HookMatcher{
			"SessionStart": {
				{
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "sh -c 'if ! command -v entire >/dev/null 2>&1; then exit 0; fi; exec entire hooks antigravity session-start'",
						},
					},
				},
			},
			"SessionEnd": {
				{
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "sh -c 'if ! command -v entire >/dev/null 2>&1; then exit 0; fi; exec entire hooks antigravity session-end'",
						},
					},
				},
			},
			"UserPromptSubmit": {
				{
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "sh -c 'if ! command -v entire >/dev/null 2>&1; then exit 0; fi; exec entire hooks antigravity user-prompt-submit'",
						},
					},
				},
			},
			"PostToolUse": {
				{
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "sh -c 'if ! command -v entire >/dev/null 2>&1; then exit 0; fi; exec entire hooks antigravity post-tool-use'",
						},
					},
				},
			},
			"Stop": {
				{
					Hooks: []HookCommand{
						{
							Type:    "command",
							Command: "sh -c 'if ! command -v entire >/dev/null 2>&1; then exit 0; fi; exec entire hooks antigravity stop'",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(hookConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks config: %w", err)
	}

	if err := os.WriteFile(hooksFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", hooksFile, err)
	}

	res := InstallHooksResponse{
		Success: true,
		Message: "Antigravity session hooks injected successfully",
		Path:    hooksFile,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(res)
}

func handleAreHooksInstalled(targetDir string) error {
	dirPath := getAntigravityDir(targetDir)
	hooksFile := filepath.Join(dirPath, "hooks.json")

	installed := false
	if info, err := os.Stat(hooksFile); err == nil && !info.IsDir() && info.Size() > 0 {
		installed = true
	} else if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
		// Check if directory exists and has any files
		entries, err := os.ReadDir(dirPath)
		if err == nil && len(entries) > 0 {
			installed = true
		}
	}

	res := InstalledResponse{
		Installed: installed,
	}

	encoder := json.NewEncoder(os.Stdout)
	return encoder.Encode(res)
}
