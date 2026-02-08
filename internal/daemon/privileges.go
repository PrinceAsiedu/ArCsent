package daemon

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func DropPrivileges(userID, groupID string) error {
	if userID == "" && groupID == "" {
		return nil
	}

	uid, err := resolveUserID(userID)
	if err != nil {
		return err
	}
	gid, err := resolveGroupID(groupID)
	if err != nil {
		return err
	}

	if gid >= 0 {
		if err := syscall.Setgid(gid); err != nil {
			return fmt.Errorf("setgid: %w", err)
		}
	}
	if uid >= 0 {
		if err := syscall.Setuid(uid); err != nil {
			return fmt.Errorf("setuid: %w", err)
		}
	}

	return nil
}

func RequirePrivilegeDrop(userID, groupID string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	if userID == "" {
		return fmt.Errorf("daemon.user must be set when running as root")
	}
	if groupID == "" {
		return fmt.Errorf("daemon.group must be set when running as root")
	}
	return nil
}

var passwdPath = "/etc/passwd"
var groupPath = "/etc/group"

func resolveUserID(value string) (int, error) {
	return resolveID("user", value, passwdPath)
}

func resolveGroupID(value string) (int, error) {
	return resolveID("group", value, groupPath)
}

func resolveID(label, value, lookupPath string) (int, error) {
	if value == "" {
		return -1, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return lookupName(label, value, lookupPath)
	}
	return parsed, nil
}

func lookupName(label, name, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return -1, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		if parts[0] != name {
			continue
		}
		id, err := strconv.Atoi(parts[2])
		if err != nil || id < 0 {
			return -1, fmt.Errorf("invalid %s id for %s in %s", label, name, path)
		}
		return id, nil
	}

	if err := scanner.Err(); err != nil {
		return -1, fmt.Errorf("read %s: %w", path, err)
	}

	return -1, fmt.Errorf("unknown %s %q", label, name)
}
