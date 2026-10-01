//go:build !windows

package auth

import "os"

func platformRestrictCredentialFile(path string) error {
	return os.Chmod(path, 0600)
}
