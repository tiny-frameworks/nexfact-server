// setup.go
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

// SetupTestEnv kopiert das zentrale testdata-Verzeichnis isoliert in ein TempDir
func SetupTestEnv(t *testing.T, provider string) (env, resources string) {
	t.Helper()

	// 1. Pfad zum zentralen testdata-Repository suchen (relativ oder per ENV)
	sourceEnv := os.Getenv("NEXFACT_TESTDATA_DIR")
	if sourceEnv == "" {
		// Relative Suche im Workspace (Projekte liegen nebeneinander)
		sourceEnv = filepath.Join("..", "..", "..", "nexfact-assets", "testdata")
	}

	if _, err := os.Stat(sourceEnv); os.IsNotExist(err) {
		t.Skipf("Skipping test: testdata repository not found at %s", sourceEnv)
	}

	// 2. Isolierter Ziel-Ordner im Go-TempDir
	targetEnv := filepath.Join(t.TempDir(), "testdata")

	// 3. Verzeichnis kopieren
	if err := copyDir(sourceEnv, targetEnv); err != nil {
		t.Fatalf("failed to copy test env: %v", err)
	}

	return setupProvider(provider, targetEnv)
}

// Hilfsfunktion: Findet einen freien Port für isolierte Tests
func GetFreeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on free port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

// Hilfsfunktion: Rekursives Kopieren von Verzeichnissen
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

// Hilfsfunktion: system.yaml abhängig von 'provider' kopieren
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
