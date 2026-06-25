package updater

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAppBundleSkipsRootDirectoryEntry(t *testing.T) {
	body := zipWithEntries(t, []zipEntry{
		{name: "GoPass.app/", mode: os.ModeDir | 0o755},
		{name: "GoPass.app/Contents/", mode: os.ModeDir | 0o755},
		{name: "GoPass.app/Contents/MacOS/", mode: os.ModeDir | 0o755},
		{name: "GoPass.app/Contents/MacOS/gopass", mode: 0o755, contents: "binary"},
		{name: "GoPass.app/Contents/Info.plist", mode: 0o644, contents: "plist"},
	})

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}

	targetApp := filepath.Join(t.TempDir(), "GoPass.app")
	if err := extractAppBundle(reader.File, "GoPass.app", targetApp); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(targetApp, "GoPass.app")); !os.IsNotExist(err) {
		t.Fatalf("root directory entry created nested app bundle, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetApp, "Contents", "MacOS", "gopass")); err != nil {
		t.Fatalf("expected app executable to be extracted: %v", err)
	}
}

type zipEntry struct {
	name     string
	mode     os.FileMode
	contents string
}

func zipWithEntries(t *testing.T, entries []zipEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, entry := range entries {
		header := &zip.FileHeader{
			Name:   entry.name,
			Method: zip.Deflate,
		}
		header.SetMode(entry.mode)
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if entry.mode.IsDir() {
			continue
		}
		if _, err := file.Write([]byte(entry.contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
