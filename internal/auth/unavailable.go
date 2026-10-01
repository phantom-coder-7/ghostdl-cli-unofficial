package auth

import (
	"errors"
	"io/fs"
	"net"
	"os/exec"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/zalando/go-keyring"
)

const (
	windowsErrNoSuchLogonSession   syscall.Errno = 1312 // ERROR_NO_SUCH_LOGON_SESSION
	windowsErrRPCServerUnavailable syscall.Errno = 1722 // RPC_S_SERVER_UNAVAILABLE
	windowsErrNotSupported         syscall.Errno = 50   // ERROR_NOT_SUPPORTED
)

// keychainUnavailable reports whether err means the OS keychain backend cannot be used
// (as opposed to a normal credential miss or a fatal keychain error).
func keychainUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return true
	}
	if strings.Contains(err.Error(), "dbus: couldn't determine address of session bus") {
		return true
	}
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) {
		switch dbusErr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner":
			return true
		}
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case windowsErrNoSuchLogonSession, windowsErrRPCServerUnavailable, windowsErrNotSupported:
			return true
		}
	}
	return false
}
