// Copyright 2026 Georg Hagn (tiny-frameworks)
// SPDX-License-Identifier: Apache-2.0

package watcher_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codeberg.org/tiny-frameworks/nexfact-server/internal/config"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/testsetup"
	"codeberg.org/tiny-frameworks/nexfact-server/internal/watcher"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/logger"
)

func TestDirectoryWatcherProcessesValidJson(t *testing.T) {
	envRoot, resourcePath := testsetup.SetupTestEnv(t, testsetup.NATIVE)
	lgr := logger.Logger.With("test_name", "dir_watcher")

	orch, err := orchestrator.New(envRoot)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	watchDir := filepath.Join(envRoot, "watch")
	processedDir := filepath.Join(envRoot, "processed")
	errorDir := filepath.Join(envRoot, "error")
	jsonFileName := "test_generate_fx.json"

	_ = os.MkdirAll(watchDir, 0755)
	_ = os.MkdirAll(processedDir, 0755)
	_ = os.MkdirAll(errorDir, 0755)

	cfg := config.Config{
		WatchDir:     watchDir,
		ProcessedDir: processedDir,
		ErrorDir:     errorDir,
		PollInterval: 50 * time.Millisecond,
	}

	w := watcher.New(cfg, orch, lgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// 1. create Test-File in watchDir
	testSource := filepath.Join(resourcePath, jsonFileName)
	testFile := filepath.Join(watchDir, jsonFileName)
	tmpFile := filepath.Join(envRoot, jsonFileName)

	atomicCopy(t, testSource, testFile, tmpFile)

	// 2.Wait until the ticker interval has taken effect.
	time.Sleep(200 * time.Millisecond)

	// 3. Check: File must have disappeared from watchDir.
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("file still exists in watch_dir, should have been moved")
	}

	// 4. Check: File must have ended up in processedDir.
	processedFile := filepath.Join(processedDir, jsonFileName)
	if _, err := os.Stat(processedFile); os.IsNotExist(err) {
		t.Errorf("file was not moved to processed_dir")
	}

	// stop Watcher
	cancel()
	w.Stop()
}

// AtomicCopy copies srcPath nach dstPath atomar.
func atomicCopy(t *testing.T, srcPath, dstPath, tmpPath string) {
	t.Helper()

	sourceFileStat, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("failed to open source file: %s", err.Error())
	}

	if !sourceFileStat.Mode().IsRegular() {
		t.Fatalf("%s is not a regular file", srcPath)
	}

	source, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("failed to open source file: %s", err.Error())
	}
	defer source.Close()

	destination, err := os.Create(tmpPath)
	if err != nil {
		t.Fatalf("failed to create destination file: %s", err.Error())
	}
	defer destination.Close()

	io.Copy(destination, source)
	if err != nil {
		t.Fatalf("failed copy to destination file: %s", err.Error())
	}

	// ATOMIC STEP: rename in targetDirectory
	if err := os.Rename(tmpPath, dstPath); err != nil {
		t.Fatalf("failed to replace target file: %s", err.Error())
	}

}
