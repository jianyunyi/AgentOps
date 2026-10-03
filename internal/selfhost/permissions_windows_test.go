//go:build windows

package selfhost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestRestrictWindowsDACL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "protected")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := Restrict(dir); err != nil {
		t.Fatal(err)
	}

	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	// DACL reports absence through err; its boolean return means defaulted.
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		t.Fatalf("DACL missing: err=%v", err)
	}
	if dacl.AceCount != 2 {
		t.Fatalf("expected exactly SYSTEM and Administrators ACEs, got %d", dacl.AceCount)
	}

	s := sd.String()
	if !strings.Contains(s, "D:P") || !strings.Contains(s, ";;;SY)") || !strings.Contains(s, ";;;BA)") {
		t.Fatalf("unexpected protected DACL: %s", s)
	}
}
