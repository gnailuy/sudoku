package recovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/privatefs"
)

func TestWriteDiscoverAndDeletePrivateRecords(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "state", "sudoku", "recovery")
	store := NewStore(directory)
	first, _ := NewID()
	second, _ := NewID()
	if err := store.Write(first, "first", []byte("session-one")); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Now().Add(time.Minute) }
	if err := store.Write(second, "second", []byte("session-two")); err != nil {
		t.Fatal(err)
	}

	if err := privatefs.ValidateDirectory(directory); err != nil {
		t.Fatalf("recovery directory is not private: %v", err)
	}
	for _, id := range []string{first, second} {
		if err := privatefs.ValidateFile(filepath.Join(directory, id+".json")); err != nil {
			t.Fatalf("recovery record is not private: %v", err)
		}
	}
	records, err := store.Discover(func(data []byte) error {
		if !strings.HasPrefix(string(data), "session-") {
			t.Fatal("validator received wrong data")
		}
		return nil
	})
	if err != nil || len(records) != 2 || records[0].ID != second {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if err := store.Delete(second); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, second+".json")); !os.IsNotExist(err) {
		t.Fatalf("record was not deleted: %v", err)
	}
}

func TestDiscoverPrunesMalformedExpiredAndRejectedRecords(t *testing.T) {
	directory := t.TempDir()
	store := NewStore(directory)
	store.now = func() time.Time { return time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC) }
	expired, _ := NewID()
	if err := store.Write(expired, "old", []byte("old")); err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
	rejected, _ := NewID()
	if err := store.Write(rejected, "bad", []byte("reject")); err != nil {
		t.Fatal(err)
	}
	malformed := filepath.Join(directory, strings.Repeat("a", 32)+".json")
	if err := os.WriteFile(malformed, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}

	records, err := store.Discover(func(data []byte) error {
		if string(data) == "reject" {
			return os.ErrInvalid
		}
		return nil
	})
	if err != nil || len(records) != 0 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	for _, path := range []string{filepath.Join(directory, expired+".json"), filepath.Join(directory, rejected+".json"), malformed} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("invalid record remains: %s", path)
		}
	}
}

func TestDiscoveryIgnoresSymlinksAndWriteRejectsSymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	store := NewStore(target)
	id, _ := NewID()
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(target, id+".json")); err != nil {
		t.Fatal(err)
	}
	records, err := store.Discover(nil)
	if err != nil || len(records) != 0 {
		t.Fatalf("records=%v err=%v", records, err)
	}

	link := filepath.Join(root, "linked-directory")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := NewStore(link).Write(id, "", []byte("session")); err == nil {
		t.Fatal("write followed a symlink directory")
	}
}

func TestDefaultDirectoryUsesPlatformApplicationData(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{name: "Linux XDG", goos: "linux", env: map[string]string{"XDG_STATE_HOME": "/state"}, home: "/home/player", want: filepath.Join("/state", "sudoku", "recovery")},
		{name: "Linux fallback", goos: "linux", home: "/home/player", want: filepath.Join("/home/player", ".local", "state", "sudoku", "recovery")},
		{name: "macOS", goos: "darwin", home: "/Users/player", want: filepath.Join("/Users/player", "Library", "Application Support", "Sudoku", "recovery")},
		{name: "Windows", goos: "windows", env: map[string]string{"LOCALAPPDATA": `C:\Users\player\AppData\Local`}, want: filepath.Join(`C:\Users\player\AppData\Local`, "Sudoku", "recovery")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, err := defaultDirectoryFor(test.goos, func(key string) string { return test.env[key] }, func() (string, error) { return test.home, nil })
			if err != nil || path != test.want {
				t.Fatalf("path=%q want=%q err=%v", path, test.want, err)
			}
		})
	}
	if _, err := defaultDirectoryFor("windows", func(string) string { return "" }, func() (string, error) { return "", nil }); err == nil {
		t.Fatal("missing LOCALAPPDATA did not fail closed")
	}
}

func TestDiscoverPrunesUnsupportedVersionAndUnsafeLabel(t *testing.T) {
	directory := t.TempDir()
	store := NewStore(directory)
	for index, mutate := range []func(map[string]any){
		func(value map[string]any) { value["version"] = float64(99) },
		func(value map[string]any) { value["source"] = "unsafe\nlabel" },
	} {
		id, _ := NewID()
		if err := store.Write(id, "safe", []byte("session")); err != nil {
			t.Fatal(err)
		}
		path := store.path(id)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatal(err)
		}
		mutate(value)
		data, _ = json.Marshal(value)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		records, err := store.Discover(nil)
		if err != nil || len(records) != 0 {
			t.Fatalf("case %d records=%v err=%v", index, records, err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("case %d invalid record remains", index)
		}
	}
}
