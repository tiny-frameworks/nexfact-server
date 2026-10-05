// Copyright 2026 Georg Hagn
// SPDX-License-Identifier: Apache-2.0

package testsetup

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
)

const ( // provider consts
	CONTAINER = "container"
	NATIVE    = "native"
)

// SetupTestEnv copies the central testdata directory into a temporary directory in isolation.
func SetupTestEnv(t *testing.T, provider string) (env, resources string) {
	t.Helper()

	// 1. Locate the path to the central testdata repository (relative or via ENV)
	sourceEnv := os.Getenv("NEXFACT_TESTDATA_DIR")
	if sourceEnv == "" {
		// Relative search within the workspace (projects are located side-by-side)
		sourceEnv = filepath.Join("..", "..", "..", "nexfact-assets", "testdata")
	}

	if _, err := os.Stat(sourceEnv); os.IsNotExist(err) {
		t.Skipf("Skipping test: testdata repository not found at %s", sourceEnv)
	}

	// 2. Isolated target folder in the Go TempDir
	targetEnv := filepath.Join(t.TempDir(), "testdata")

	// 3. copy directory
	if err := copyDir(sourceEnv, targetEnv); err != nil {
		t.Fatalf("failed to copy test env: %v", err)
	}

	return setupProvider(provider, targetEnv)
}

// Helper function: Finds a free port for isolated tests.
func GetFreeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on free port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

// Helper function: Recursive directory copying
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Helper function: Copy system.yaml based on 'provider'
func setupProvider(provider string, targetDir string) (envRoot string, resourcePath string) {
	envRoot, _ = filepath.Abs(filepath.Join(targetDir, "env"))
	resourcePath, _ = filepath.Abs(filepath.Join(targetDir, "resources"))

	systemNativeResource := filepath.Join(resourcePath, "system.native")
	systemContainerResource := filepath.Join(resourcePath, "system.container")
	systemYamlTarget := filepath.Join(envRoot, "system", "system.yaml")

	switch provider {
	case NATIVE:
		copyFile(systemNativeResource, systemYamlTarget)
	case CONTAINER:
		copyFile(systemContainerResource, systemYamlTarget)
	}

	return envRoot, resourcePath
}
