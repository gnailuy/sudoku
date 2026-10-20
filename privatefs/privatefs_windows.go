//go:build windows

// Package privatefs enforces platform-native private file and directory storage.
package privatefs

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var allowedSystemSIDs = map[string]struct{}{
	"S-1-5-18":     {}, // Local System
	"S-1-5-32-544": {}, // Built-in Administrators
}

// ValidateFile requires a regular, non-reparse-point file whose protected DACL
// grants access only to the current user, Local System, and Administrators.
func ValidateFile(path string) error {
	if err := validateKind(path, false); err != nil {
		return err
	}
	return validateACL(path)
}

// ProtectFile replaces inherited permissions with a private Windows DACL.
func ProtectFile(path string) error {
	if err := validateKind(path, false); err != nil {
		return err
	}
	if err := protect(path, false); err != nil {
		return err
	}
	return ValidateFile(path)
}

// ValidateDirectory requires a normal directory with the same private DACL.
func ValidateDirectory(path string) error {
	if err := validateKind(path, true); err != nil {
		return err
	}
	return validateACL(path)
}

// ProtectDirectory replaces inherited permissions and makes child objects
// inherit the private DACL.
func ProtectDirectory(path string) error {
	if err := validateKind(path, true); err != nil {
		return err
	}
	if err := protect(path, true); err != nil {
		return err
	}
	return ValidateDirectory(path)
}

func validateKind(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	attributes, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(path))
	if err != nil {
		return err
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return errors.New("reparse points are not private storage")
	}
	if directory != info.IsDir() || (!directory && !info.Mode().IsRegular()) {
		return errors.New("unexpected private storage object type")
	}
	return nil
}

func currentUserSID() (*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	return user.User.Sid, nil
}

func protect(path string, directory bool) error {
	user, err := currentUserSID()
	if err != nil {
		return fmt.Errorf("resolve current Windows user: %w", err)
	}
	inheritance := ""
	if directory {
		inheritance = "OICI"
	}
	sddl := fmt.Sprintf("D:P(A;%s;FA;;;%s)(A;%s;FA;;;SY)(A;%s;FA;;;BA)", inheritance, user.String(), inheritance, inheritance)
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return fmt.Errorf("build private Windows DACL: %w", err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return fmt.Errorf("read private Windows DACL: %w", err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		return fmt.Errorf("apply private Windows DACL: %w", err)
	}
	return nil
}

func validateACL(path string) error {
	user, err := currentUserSID()
	if err != nil {
		return fmt.Errorf("resolve current Windows user: %w", err)
	}
	allowed := map[string]struct{}{user.String(): {}}
	for sid := range allowedSystemSIDs {
		allowed[sid] = struct{}{}
	}
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read Windows DACL: %w", err)
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return fmt.Errorf("read Windows DACL control: %w", err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return errors.New("Windows DACL inherits permissions")
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil || dacl.AceCount == 0 {
		return errors.New("Windows DACL is missing")
	}
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil {
			return fmt.Errorf("read Windows DACL entry: %w", err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("Windows DACL contains an unsupported entry")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.IsValid() {
			return errors.New("Windows DACL contains an invalid principal")
		}
		if _, ok := allowed[sid.String()]; !ok {
			return fmt.Errorf("Windows DACL grants access to %s", sid.String())
		}
	}
	return nil
}
