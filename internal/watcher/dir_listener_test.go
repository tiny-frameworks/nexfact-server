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
	"codeberg.org/tiny-frameworks/nexfact-server/internal/watcher"
	"codeberg.org/tiny-frameworks/nexfact/orchestrator"
	"codeberg.org/tiny-frameworks/nexutils/logger"
)

func TestDirectoryWatcherProcessesValidJson(t *testing.T) {
	tempDir := t.TempDir()
	watchDir := filepath.Join(tempDir, "watch")
	processedDir := filepath.Join(tempDir, "processed")
	errorDir := filepath.Join(tempDir, "error")
	jsonFileName := "test_generate_fx.json"

	_ = os.MkdirAll(watchDir, 0755)
	_ = os.MkdirAll(processedDir, 0755)
	_ = os.MkdirAll(errorDir, 0755)

	lgr := logger.Logger.With("test", "dir_watcher")

	testEnvParent := getTestEnvParent(t)

	env := filepath.Join(testEnvParent, "env")
	orch, err := orchestrator.New(env)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

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

	// 1. Test-Datei im watchDir erstellen
	testSource := filepath.Join(testEnvParent, "resources", jsonFileName)
	testFile := filepath.Join(watchDir, jsonFileName)
	tmpFile := filepath.Join(tempDir, jsonFileName)

	atomicCopy(t, testSource, testFile, tmpFile)

	// 2. Warten bis der Ticker-Intervall gegriffen hat
	time.Sleep(200 * time.Millisecond)

	// 3. Prüfen: Datei muss aus watchDir verschwunden sein
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("file still exists in watch_dir, should have been moved")
	}

	// 4. Prüfen: Datei muss im processedDir gelandet sein
	processedFile := filepath.Join(processedDir, jsonFileName)
	if _, err := os.Stat(processedFile); os.IsNotExist(err) {
		t.Errorf("file was not moved to processed_dir")
	}

	// Watcher beenden
	cancel()
	w.Stop()
}

func getTestEnvParent(t *testing.T) string {
	t.Helper()

	// 1. Prüfe ob eine ENV-Variable gesetzt ist (z.B. in go.work / lokalen Tests)
	if envPath := os.Getenv("NEXFACT_TEST_ENV_PARENT"); envPath != "" {
		return envPath
	}

	// 2. Fallback auf lokales lüttes testdata Ordnerchen im Repo
	localPath := filepath.Join("testdata", "testenv")
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	// 3. Fallback auf relatives nexfact Nachbar-Repo
	relPath := filepath.Join("..", "..", "..", "nexfact", "orchestrator", "testdata")
	absFile, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatalf("Can not build absolute Path for %s", relPath)
	}
	if _, err := os.Stat(absFile); err == nil {
		return absFile
	}

	t.Fatalf("No valid env found for testing")
	return ""
}

// AtomicCopy kopiert srcPath nach dstPath atomar.
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

	// ATOMARER SCHRITT: Umbenennen im Zielverzeichnis
	if err := os.Rename(tmpPath, dstPath); err != nil {
		t.Fatalf("failed to replace target file: %s", err.Error())
	}

}
