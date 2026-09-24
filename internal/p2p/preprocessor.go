// Copyright 2026 Georg Hagn
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codeberg.org/tiny-frameworks/nexutils/errors"
)

// PreProcess takes the incoming RPC JSON, writes any Base64 files or URLs found
// to tempDir, and returns the JSON adapted for the orchestrator.
func PreProcess(tempDir string, rawJSON []byte) ([]byte, error) {

	var data map[string]any

	if err := json.Unmarshal(rawJSON, &data); err != nil {
		return nil,
			errors.Wrap(
				errors.InvalidValue,
				fmt.Sprint("preprocessor: invalid json"),
				"orchestrator.internal.preprocessor.PreProcess",
				err,
			)
	}

	options, ok := data["options"].(map[string]any)
	if !ok {
		// No options available? Then pass the JSON through unchanged.
		return rawJSON, nil
	}

	for key, value := range options {
		switch key {
		case "attachments":
			continue // Processed in the separate attachments block.
		case "queue":
			// All pure option values ​​that are DEFINITELY NOT files end up here.
			// Simply ignore them; they remain as they are.
			continue
		default:
			// Attempts to dynamically save a file (Base64 map) in the tempDir.
			path, err := handleFileField(tempDir, value)
			if err != nil {
				return nil, err
			}
			if path != "" {
				options[key] = path // Replaces the object with the local file path
			}

		}
	}

	// 2. check for Attachments-Array
	if rawAttachments, exists := options["attachments"]; exists {
		if attList, ok := rawAttachments.([]any); ok {
			var newPaths []string
			for _, item := range attList {
				path, err := handleFileField(tempDir, item)
				if err != nil {
					return nil, err
				}
				if path != "" {
					newPaths = append(newPaths, path)
				}
			}
			options["attachments"] = newPaths // Replaces the list with path strings.
		}
	}

	// Back to Byte-Stream for orchestrator.RunJsonJob
	return json.Marshal(data)
}

// handleFileField saves Base64 bytes or downloads as a local file.
func handleFileField(tempDir string, rawField any) (string, error) {
	fileMap, ok := rawField.(map[string]any)
	if !ok {
		// No map object? Then it is either already a path string
		// or a primitive value (int, bool). Leave unchanged:
		return "", nil
	}

	fileName, _ := fileMap["name"].(string)
	if fileName == "" {
		fileName = "unnamed_file"
	}
	targetPath := filepath.Join(tempDir, filepath.Base(fileName))

	// Case A: Raw Data (Base64-String from JSON)
	if dataStr, ok := fileMap["data"].(string); ok && dataStr != "" {
		bytes, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return "", errors.Wrap(
				errors.InvalidValue,
				fmt.Sprintf("base64 decode failed for %s", fileName),
				"orchestrator.internal.preprocessor.PreProcess",
				err,
			)
		}
		if err := os.WriteFile(targetPath, bytes, 0600); err != nil {
			return "", errors.Wrap(
				errors.WriteError,
				fmt.Sprintf("write failed for %s", targetPath),
				"orchestrator.internal.preprocessor.PreProcess",
				err)
		}
		return targetPath, nil
	}

	// Case B: Load file via URL
	if urlStr, ok := fileMap["url"].(string); ok && urlStr != "" {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(urlStr)
		if err != nil {
			return "", errors.Wrap(
				errors.InvalidValue,
				fmt.Sprintf("download failed for %s", urlStr),
				"orchestrator.internal.preprocessor.PreProcess",
				err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", errors.New(
				errors.InvalidValue,
				fmt.Sprintf("download from %s returned status %d", urlStr, resp.StatusCode),
				"orchestrator.internal.preprocessor.PreProcess")
		}

		out, err := os.Create(targetPath)
		if err != nil {
			return "", errors.Wrap(
				errors.InvalidValue,
				fmt.Sprintf("failed to create stagedfile"),
				"orchestrator.internal.preprocessor.PreProcess",
				err)
		}
		defer out.Close()

		if _, err := io.Copy(out, resp.Body); err != nil {
			return "", errors.Wrap(
				errors.InvalidValue,
				fmt.Sprintf("failed to save stream to %s", targetPath),
				"orchestrator.internal.preprocessor.PreProcess",
				err)
		}

		return targetPath, nil
	}

	return "", errors.New(
		errors.InvalidValue,
		fmt.Sprintf("invalid input file %s: neither Data nor URL provided", fileMap["name"]),
		"orchestrator.internal.preprocessor.PreProcess")
}
