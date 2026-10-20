package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnailuy/sudoku/game"
	"github.com/gnailuy/sudoku/privatefs"
)

func TestLoadManagedGuestKeyPersistsOwnerOnlyKey(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("LOCALAPPDATA", dataHome)

	first, err := loadManagedGuestKey()
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadManagedGuestKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 32 || !bytes.Equal(first, second) {
		t.Fatalf("managed key was not stable: first=%d bytes equal=%v", len(first), bytes.Equal(first, second))
	}
	path := filepath.Join(dataHome, "sudoku", managedGuestKeyFile)
	if err := privatefs.ValidateFile(path); err != nil {
		t.Fatalf("managed key is not private: %v", err)
	}

	if err := makeFileBroadlyReadable(path); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManagedGuestKey(); err == nil || !strings.Contains(err.Error(), "private file") {
		t.Fatalf("unsafe managed key error = %v", err)
	}
}

func TestLoadWebRuntimeSupportsGuestOnlyConfigWithoutAccounts(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	path := filepath.Join(t.TempDir(), "web.json")
	body := `{"guest":{"active_key_id":"operator-v1","keys":{"operator-v1":"` + key + `"},"lifetime_seconds":86400}}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	options := game.NewDefaultOptions(solverStore)
	options.StrategySolverKeys = solverStore.GetAllStrategySolverKeys()
	guest, account, closeRuntime, err := loadWebRuntime(context.Background(), path, filepath.Join(t.TempDir(), "sudoku.db"), options)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := closeRuntime(); err != nil {
			t.Errorf("close runtime: %v", err)
		}
	}()
	if guest == nil || guest.Documents == nil || guest.Create == nil {
		t.Fatal("guest runtime is incomplete")
	}
	if account != nil {
		t.Fatal("guest-only configuration unexpectedly enabled accounts")
	}
}

func TestReadWebConfigRejectsAccountWithoutGuestAndUnsafeFile(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	body := `{"account":{"csrf_key":"` + key + `","google":{"client_id":"client","client_secret":"secret","redirect_url":"https://client.example/callback"},"return_path_prefix":"/","cookie_path":"/","session_idle_seconds":3600,"session_absolute_seconds":7200}}`
	path := filepath.Join(t.TempDir(), "web.json")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	options := game.NewDefaultOptions(solverStore)
	_, _, _, err := loadWebRuntime(context.Background(), path, "", options)
	if err == nil || !strings.Contains(err.Error(), "requires a complete guest section") {
		t.Fatalf("account-only error = %v", err)
	}
	if err := makeFileBroadlyReadable(path); err != nil {
		t.Fatal(err)
	}
	if _, err := readWebConfig(path); err == nil || !strings.Contains(err.Error(), "private file") {
		t.Fatalf("unsafe config error = %v", err)
	}
}

func TestValidateAccountConfigRejectsPartialPolicy(t *testing.T) {
	valid := accountConfigFile{
		CSRFKey:          base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)),
		Google:           googleConfigFile{ClientID: "client", ClientSecret: "secret", RedirectURL: "https://client.example/callback"},
		ReturnPathPrefix: "/", CookiePath: "/", SessionIdleSeconds: int64(time.Hour / time.Second), SessionAbsoluteSeconds: int64((2 * time.Hour) / time.Second),
	}
	if err := validateAccountConfig(valid); err != nil {
		t.Fatal(err)
	}
	valid.Google.ClientSecret = ""
	if err := validateAccountConfig(valid); err == nil {
		t.Fatal("partial Google configuration was accepted")
	}
}
