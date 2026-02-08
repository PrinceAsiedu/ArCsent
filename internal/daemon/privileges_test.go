package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDropPrivilegesRejectsNonNumeric(t *testing.T) {
	if err := DropPrivileges("user", ""); err == nil {
		t.Fatalf("expected error for non-numeric user id")
	}
}

func TestResolveUserAndGroupByName(t *testing.T) {
	dir := t.TempDir()
	passwd := filepath.Join(dir, "passwd")
	group := filepath.Join(dir, "group")

	if err := os.WriteFile(passwd, []byte("daemon:x:1234:1234::/nonexistent:/bin/false\n"), 0o600); err != nil {
		t.Fatalf("write passwd: %v", err)
	}
	if err := os.WriteFile(group, []byte("daemon:x:1234:\n"), 0o600); err != nil {
		t.Fatalf("write group: %v", err)
	}

	origPasswd := passwdPath
	origGroup := groupPath
	passwdPath = passwd
	groupPath = group
	t.Cleanup(func() {
		passwdPath = origPasswd
		groupPath = origGroup
	})

	uid, err := resolveUserID("daemon")
	if err != nil || uid != 1234 {
		t.Fatalf("expected uid 1234, got %d (err=%v)", uid, err)
	}
	gid, err := resolveGroupID("daemon")
	if err != nil || gid != 1234 {
		t.Fatalf("expected gid 1234, got %d (err=%v)", gid, err)
	}
}
