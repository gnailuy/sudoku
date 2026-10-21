//go:build windows

package privatefs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestProtectAndValidateNativeObjects(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "private")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ProtectDirectory(directory); err != nil {
		t.Fatalf("protect directory: %v", err)
	}
	if err := ValidateDirectory(directory); err != nil {
		t.Fatalf("validate protected directory: %v", err)
	}
	file := filepath.Join(directory, "secret")
	if err := os.WriteFile(file, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ProtectFile(file); err != nil {
		t.Fatalf("protect file: %v", err)
	}
	if err := ValidateFile(file); err != nil {
		t.Fatalf("validate protected file: %v", err)
	}
}

func TestValidateRejectsInheritedAndAdditionalPrincipals(t *testing.T) {
	for _, test := range []struct {
		name      string
		protected bool
		extraACE  string
	}{
		{name: "inherited DACL", protected: false},
		{name: "additional principal", protected: true, extraACE: "(A;;FR;;;WD)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "secret")
			if err := os.WriteFile(file, []byte("secret"), 0600); err != nil {
				t.Fatal(err)
			}
			user, err := currentUserSID()
			if err != nil {
				t.Fatal(err)
			}
			prefix := "D:"
			if test.protected {
				prefix = "D:P"
			}
			sddl := fmt.Sprintf("%s(A;;FA;;;%s)(A;;FA;;;SY)(A;;FA;;;BA)%s", prefix, user.String(), test.extraACE)
			setTestDACL(t, file, sddl, test.protected)
			if err := ValidateFile(file); err == nil {
				t.Fatal("unsafe DACL unexpectedly validated")
			}
		})
	}
}

func TestRejectsReparsePointsAndWrongObjectTypes(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDirectory(file); err == nil {
		t.Fatal("file unexpectedly validated as a directory")
	}
	if err := ValidateFile(directory); err == nil {
		t.Fatal("directory unexpectedly validated as a file")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Skipf("creating a Windows symbolic link requires host permission: %v", err)
	}
	if err := ValidateFile(link); err == nil {
		t.Fatal("reparse point unexpectedly validated")
	}
	if err := ProtectFile(link); err == nil {
		t.Fatal("reparse point unexpectedly protected")
	}
}

func setTestDACL(t *testing.T, path, sddl string, protected bool) {
	t.Helper()
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	info := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION)
	if protected {
		info |= windows.PROTECTED_DACL_SECURITY_INFORMATION
	} else {
		info |= windows.UNPROTECTED_DACL_SECURITY_INFORMATION
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, info, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
}
