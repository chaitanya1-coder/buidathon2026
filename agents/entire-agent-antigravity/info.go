package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func handleInfo() error {
	agentName := "antigravity"
	if strings.Contains(filepath.Base(os.Args[0]), "opencode") {
		agentName = "opencode"
	}

	info := InfoResponse{
		Name:        agentName,
		Version:     "0.1.0",
		Description: "Entire CLI adapter for Antigravity Agent",
		Features: map[string]bool{
			"hooks":               true,
			"transcripts":         true,
			"compact_transcripts": true,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}
