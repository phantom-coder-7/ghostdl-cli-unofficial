package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

func withTestConfigDir(t *testing.T) string {
	dir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, dir)
	return dir
}

func restoreKeyringFns(t *testing.T) {
	oldSet, oldGet, oldDel := setKeyring, getKeyring, deleteKeyring
	oldRestrict := restrictCredentialFile
	t.Cleanup(func() {
		setKeyring, getKeyring, deleteKeyring = oldSet, oldGet, oldDel
		restrictCredentialFile = oldRestrict
	})
}

func captureWarnf(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := warnf
	warnf = func(format string, args ...any) {
		fmt.Fprintf(buf, format, args...)
	}
	t.Cleanup(func() { warnf = old })
	return buf
}

func TestSave_KeyringSuccessClearsLeftoverFile(t *testing.T) {
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)
	login := &LoginData{Kind: KindToken, Token: "tok"}
	b, _ := json.Marshal(login)
	if err := os.WriteFile(filePath, b, 0600); err != nil {
		t.Fatal(err)
	}

	setKeyring = func(_ string, _ string, _ string) error { return nil }

	store := &Store{}
	if err := store.Save(login); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected leftover file removed, stat err=%v", err)
	}
}

func TestSave_UnavailableWritesFileAndGetRoundTrips(t *testing.T) {
	restoreKeyringFns(t)
	warnBuf := captureWarnf(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrUnsupportedPlatform
	}
	getKeyring = func(_ string, _ string) (string, error) {
		return "", keyring.ErrUnsupportedPlatform
	}

	login := &LoginData{Kind: KindToken, Token: "secret"}
	store := &Store{AllowCredentialFile: true}
	if err := store.Save(login); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatal(err)
	}

	warn := warnBuf.String()
	if !strings.Contains(warn, "OS keychain was unavailable") {
		t.Fatalf("expected keychain warning, got %q", warn)
	}
	if !strings.Contains(warn, "user-only file") {
		t.Fatalf("expected user-only file in warning, got %q", warn)
	}
	if strings.Contains(warn, "0600") {
		t.Fatalf("warning should not claim mode 0600 on every OS, got %q", warn)
	}
	if !strings.Contains(warn, filePath) {
		t.Fatalf("expected path %q in warning, got %q", filePath, warn)
	}

	warnBuf.Reset()
	got, err := store.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != login.Token {
		t.Fatalf("token mismatch: %q vs %q", got.Token, login.Token)
	}
	if warnBuf.Len() != 0 {
		t.Fatalf("Get should not warn, got %q", warnBuf.String())
	}
}

func TestSave_UnavailableFileModeUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file permissions")
	}
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrUnsupportedPlatform
	}

	login := &LoginData{Kind: KindToken, Token: "secret"}
	store := &Store{AllowCredentialFile: true}
	if err := store.Save(login); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
	}
}

func TestSave_UnavailableFileModeWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows file permissions")
	}
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrUnsupportedPlatform
	}
	getKeyring = func(_ string, _ string) (string, error) {
		return "", keyring.ErrUnsupportedPlatform
	}

	login := &LoginData{Kind: KindToken, Token: "secret"}
	store := &Store{AllowCredentialFile: true}
	if err := store.Save(login); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != login.Token {
		t.Fatalf("token mismatch: %q vs %q", got.Token, login.Token)
	}
}

func TestSave_RestrictFailureRemovesFile(t *testing.T) {
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrUnsupportedPlatform
	}
	restrictCredentialFile = func(_ string) error {
		return errors.New("restrict failed")
	}

	err := (&Store{AllowCredentialFile: true}).Save(&LoginData{Kind: KindToken, Token: "secret"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "restrict failed") {
		t.Fatalf("expected restrict error, got %v", err)
	}
	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatal("credential file should not exist after restrict failure")
	}
}

func TestSave_KeyringSuccessDoesNotWarn(t *testing.T) {
	restoreKeyringFns(t)
	warnBuf := captureWarnf(t)
	withTestConfigDir(t)

	setKeyring = func(_ string, _ string, _ string) error { return nil }

	if err := (&Store{}).Save(&LoginData{Kind: KindToken, Token: "tok"}); err != nil {
		t.Fatal(err)
	}
	if warnBuf.Len() != 0 {
		t.Fatalf("keyring save should not warn, got %q", warnBuf.String())
	}
}

func TestSave_LockedCollectionDoesNotWriteFile(t *testing.T) {
	restoreKeyringFns(t)
	withTestConfigDir(t)

	setKeyring = func(_ string, _ string, _ string) error {
		return errors.New("failed to unlock correct collection")
	}

	store := &Store{}
	err := store.Save(&LoginData{Kind: KindToken, Token: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "--allow-credential-file") {
		t.Fatalf("locked collection must not mention --allow-credential-file, got %v", err)
	}
	path, _ := credentialFilePath()
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("credential file should not exist")
	}
}

func TestSave_DataTooBigDoesNotWriteFile(t *testing.T) {
	restoreKeyringFns(t)
	withTestConfigDir(t)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrSetDataTooBig
	}

	store := &Store{}
	err := store.Save(&LoginData{Kind: KindToken, Token: "x"})
	if !errors.Is(err, keyring.ErrSetDataTooBig) {
		t.Fatalf("expected ErrSetDataTooBig, got %v", err)
	}
	path, _ := credentialFilePath()
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatal("credential file should not exist")
	}
}

func TestGet_PrefersKeyringWhenBothExist(t *testing.T) {
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	fileLogin := &LoginData{Kind: KindToken, Token: "from-file"}
	b, _ := json.Marshal(fileLogin)
	if err := os.WriteFile(filePath, b, 0600); err != nil {
		t.Fatal(err)
	}

	getKeyring = func(_ string, _ string) (string, error) {
		kr := &LoginData{Kind: KindToken, Token: "from-keyring"}
		raw, _ := json.Marshal(kr)
		return string(raw), nil
	}
	setKeyring = func(_ string, _ string, _ string) error { return nil }

	got, err := (&Store{}).Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "from-keyring" {
		t.Fatalf("expected keyring value, got %q", got.Token)
	}
}

func TestDelete_RemovesFile(t *testing.T) {
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)
	if err := os.WriteFile(filePath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	deleteKeyring = func(_ string, _ string) error { return keyring.ErrNotFound }

	if err := (&Store{}).Delete(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, err=%v", err)
	}
}

func TestSave_UnavailableWithoutOptInDoesNotWriteFile(t *testing.T) {
	restoreKeyringFns(t)
	dir := withTestConfigDir(t)
	filePath := filepath.Join(dir, keyLogin)

	setKeyring = func(_ string, _ string, _ string) error {
		return keyring.ErrUnsupportedPlatform
	}

	err := (&Store{}).Save(&LoginData{Kind: KindToken, Token: "secret"})
	if !errors.Is(err, ErrCredentialFileOptIn) {
		t.Fatalf("expected ErrCredentialFileOptIn, got %v", err)
	}
	if !strings.Contains(err.Error(), "--allow-credential-file") {
		t.Fatalf("expected --allow-credential-file in error, got %v", err)
	}
	if _, statErr := os.Stat(filePath); !os.IsNotExist(statErr) {
		t.Fatal("credential file should not exist")
	}
}

func TestMissingKeychainError(t *testing.T) {
	restoreKeyringFns(t)
	withTestConfigDir(t)

	t.Run("unavailable", func(t *testing.T) {
		getKeyring = func(_ string, _ string) (string, error) {
			return "", keyring.ErrUnsupportedPlatform
		}
		err := (&Store{}).MissingKeychainError()
		if !errors.Is(err, ErrCredentialFileOptIn) {
			t.Fatalf("expected ErrCredentialFileOptIn, got %v", err)
		}
		if !strings.Contains(err.Error(), "--allow-credential-file") {
			t.Fatalf("expected --allow-credential-file in error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		getKeyring = func(_ string, _ string) (string, error) {
			return "", keyring.ErrNotFound
		}
		if err := (&Store{}).MissingKeychainError(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("success", func(t *testing.T) {
		getKeyring = func(_ string, _ string) (string, error) {
			return `{"kind":"token","token":"x"}`, nil
		}
		if err := (&Store{}).MissingKeychainError(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("locked", func(t *testing.T) {
		getKeyring = func(_ string, _ string) (string, error) {
			return "", errors.New("failed to unlock correct collection")
		}
		if err := (&Store{}).MissingKeychainError(); err != nil {
			t.Fatalf("locked collection should not be the opt-in error, got %v", err)
		}
	})

	t.Run("opt in", func(t *testing.T) {
		getKeyring = func(_ string, _ string) (string, error) {
			return "", keyring.ErrUnsupportedPlatform
		}
		if err := (&Store{AllowCredentialFile: true}).MissingKeychainError(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestKeychainUnavailable(t *testing.T) {
	dialErr := &net.OpError{Op: "dial", Net: "unix", Err: errors.New("connection refused")}
	securityMissing := &fs.PathError{Op: "fork/exec", Path: "/usr/bin/security", Err: fs.ErrNotExist}
	securityNoExec := &fs.PathError{Op: "fork/exec", Path: "/usr/bin/security", Err: fs.ErrPermission}
	exit36Err := exec.Command("sh", "-c", "exit 36").Run()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unsupported platform", keyring.ErrUnsupportedPlatform, true},
		{"no session bus", errors.New("dbus: couldn't determine address of session bus"), true},
		{"exec err not found", exec.ErrNotFound, true},
		{"wrapped exec err not found", fmt.Errorf("dbus-launch: %w", exec.ErrNotFound), true},
		{"dial op error", dialErr, true},
		{"service unknown", dbus.Error{Name: "org.freedesktop.DBus.Error.ServiceUnknown", Body: []interface{}{"msg"}}, true},
		{"name has no owner", dbus.Error{Name: "org.freedesktop.DBus.Error.NameHasNoOwner", Body: []interface{}{"msg"}}, true},
		{"err not found", keyring.ErrNotFound, false},
		{"data too big", keyring.ErrSetDataTooBig, false},
		{"unlock collection", errors.New("failed to unlock correct collection"), false},
		{"other dbus", dbus.Error{Name: "org.freedesktop.Secret.Service.Error.IsLocked", Body: []interface{}{"msg"}}, false},
		{"security missing", securityMissing, true},
		{"security not executable", securityNoExec, true},
		{"security exit status", exit36Err, false},
		{"exit status string", fmt.Errorf("exit status 36"), false},
		{"windows no logon session", syscall.Errno(1312), true},
		{"windows rpc unavailable", syscall.Errno(1722), true},
		{"windows not supported", syscall.Errno(50), true},
		{"windows element not found", syscall.Errno(1168), false},
		{"windows invalid param", syscall.Errno(87), false},
		{"windows bad username", syscall.Errno(2202), false},
		{"windows access denied", syscall.Errno(5), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := keychainUnavailable(tc.err); got != tc.want {
				t.Fatalf("keychainUnavailable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
