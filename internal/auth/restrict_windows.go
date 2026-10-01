//go:build windows

package auth

import "golang.org/x/sys/windows"

// fileAllAccess is Win32 FILE_ALL_ACCESS (icacls F on files), not GENERIC_ALL.
const fileAllAccess = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x1FF

// platformRestrictCredentialFile sets an owner-only DACL on path: GRANT to the
// current user SID only, with PROTECTED_DACL (no inheritance). Same model as
// Unix 0600 and OpenSSH user private keys — not OpenSSH host keys (SYSTEM and
// Administrators). Host-key ACLs are wrong here: an unelevated member of
// Administrators still runs with a UAC-filtered token and is not in that DACL,
// which is why users cannot open host keys without elevation. login-data must
// stay readable/writable by the account that ran ghostdl auth login; adding SYSTEM
// or Administrators would not match that intent.
func platformRestrictCredentialFile(path string) error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer token.Close()

	tu, err := token.GetTokenUser()
	if err != nil {
		return err
	}

	entries := []windows.EXPLICIT_ACCESS{{
		AccessPermissions: fileAllAccess,
		AccessMode:        windows.GRANT_ACCESS,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(tu.User.Sid),
		},
	}}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
}
