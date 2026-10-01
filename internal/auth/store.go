package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	service  = "ghostdl-unofficial"
	keyLogin = "login-data"
)

var (
	setKeyring    = keyring.Set
	getKeyring    = keyring.Get
	deleteKeyring = keyring.Delete
	// restrictCredentialFile hardens login-data after WriteFile (Unix 0600 / Windows owner-only DACL).
	// Overridable in tests.
	restrictCredentialFile = platformRestrictCredentialFile
	// warnf prints store warnings (credential-file fallback). Overridable in tests.
	warnf = func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format, args...)
	}
)

// ErrCredentialFileOptIn is returned when the OS keychain cannot be reached
// and the caller has not opted in to the credential file.
var ErrCredentialFileOptIn = errors.New("OS keychain is unavailable. Run ghostdl auth login --allow-credential-file to store the pairing token in the credential file login-data in your config directory. That file is limited to your user account and is weaker than the OS keychain.")

// Store wraps read/write operations on credentials.
// Save writes the OS keychain when that service accepts the token.
// The credential file login-data is written only when AllowCredentialFile is set
// and the OS keychain cannot be reached.
type Store struct {
	AllowCredentialFile bool
}

func NewStore() (*Store, error) {
	return &Store{}, nil
}

func credentialFilePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, keyLogin), nil
}

func (s *Store) Save(data *LoginData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = setKeyring(service, keyLogin, string(b))
	if err == nil {
		if path, perr := credentialFilePath(); perr == nil {
			_ = os.Remove(path)
		}
		return nil
	}
	if !keychainUnavailable(err) {
		return err
	}
	if !s.AllowCredentialFile {
		return ErrCredentialFileOptIn
	}
	path, err := credentialFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return err
	}
	if err := restrictCredentialFile(path); err != nil {
		_ = os.Remove(path)
		return err
	}
	warnf("Warning: OS keychain was unavailable; pairing token stored in a user-only file: %s\n", path)
	return nil
}

// MissingKeychainError reports whether login should stop before pairing.
// It probes the OS keychain directly and does not read an existing login-data file.
// A reachable keychain, including one that refuses access, returns nil.
func (s *Store) MissingKeychainError() error {
	if s.AllowCredentialFile {
		return nil
	}
	_, err := getKeyring(service, keyLogin)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if keychainUnavailable(err) {
		return ErrCredentialFileOptIn
	}
	return nil
}

func (s *Store) Get() (*LoginData, error) {
	raw, err := getKeyring(service, keyLogin)
	if err == nil {
		var data LoginData
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return nil, err
		}
		return &data, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) && !keychainUnavailable(err) {
		return nil, err
	}
	path, pathErr := credentialFilePath()
	if pathErr != nil {
		return nil, err
	}
	b, fileErr := os.ReadFile(path)
	if fileErr != nil {
		if os.IsNotExist(fileErr) {
			return nil, err
		}
		return nil, fileErr
	}
	var data LoginData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (s *Store) Delete() error {
	if path, err := credentialFilePath(); err == nil {
		_ = os.Remove(path)
	}
	err := deleteKeyring(service, keyLogin)
	if err == nil || errors.Is(err, keyring.ErrNotFound) || keychainUnavailable(err) {
		return nil
	}
	return err
}

func (s *Store) IsLoggedIn() bool {
	_, err := s.Get()
	return err == nil
}
