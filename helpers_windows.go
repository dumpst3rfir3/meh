//go:build windows

package meh

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	endpoint "github.com/preludeorg/libraries/go/tests/endpoint"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	procFindWindowW  = user32dll.NewProc("FindWindowW")
	procPostMessageW = user32dll.NewProc("PostMessageW")
	user32dll        = windows.NewLazyDLL("user32.dll")
)

// If the test opens a window, this can be used to minimize it.
// If running tests automatically from the Prelude Detect platform,
// this is not needed since no window will be opened. This is
// more for running manual, standalone tests
func MinimizeWindow(name string) {
	var e error
	var stat uintptr
	var title *uint16
	var wh uintptr

	endpoint.Wait(time.Second)
	endpoint.Say("Minimizing %s window", name)

	title, _ = windows.UTF16PtrFromString(name)
	wh, _, _ = procFindWindowW.Call(
		0,
		uintptr(unsafe.Pointer(title)),
	)
	stat, _, e = procPostMessageW.Call(
		uintptr(wh),
		0x0112, // wmsyscommand
		0xf020, // scminimize,
		0,
	)

	if stat == 0 {
		endpoint.Say("Failed to minimize %s window: %s", name, e)
	}
}

// Parses a registry path string to extract the root - this is
// used by the regDelete function
func parseRegPath(str string) (registry.Key, string, error) {
	var path []string
	var root registry.Key

	str = filepath.Clean(str)

	// Split for parsing
	path = strings.Split(filepath.ToSlash(str), "/")
	if len(path) == 0 {
		return root, "", fmt.Errorf("empty key provided")
	}

	// Determine registry root
	switch strings.ToLower(path[0]) {
	case "hkcc":
		root = registry.CURRENT_CONFIG
	case "hkcr":
		root = registry.CLASSES_ROOT
	case "hkcu":
		root = registry.CURRENT_USER
	case "hklm":
		root = registry.LOCAL_MACHINE
	case "hkpd":
		root = registry.PERFORMANCE_DATA
	case "hku":
		root = registry.USERS
	default:
		return root, "", fmt.Errorf("invalid key: %s", str)
	}

	// Return parsed components
	return root, filepath.Join(path[1:]...), nil
}

func procAttrs(cmd []string) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CmdLine: strings.Join(cmd, " "),
	}
}

// Delete a registry key
func RegDelete(key string) error {
	var e error
	var k registry.Key
	var path string
	var perms uint32 = registry.ENUMERATE_SUB_KEYS
	var root registry.Key
	var subkeys []string

	if root, path, e = parseRegPath(key); e != nil {
		return e
	}

	// Get key
	if k, e = registry.OpenKey(root, path, perms); e != nil {
		return fmt.Errorf("failed to get key %s: %w", key, e)
	}
	defer k.Close()

	// Read subkeys
	if subkeys, e = k.ReadSubKeyNames(0); e != nil {
		return fmt.Errorf(
			"failed to read subkeys for key %s: %w",
			key,
			e,
		)
	}

	k.Close()

	for _, subkey := range subkeys {
		if e = RegDelete(filepath.Join(key, subkey)); e != nil {
			return e
		}
	}

	endpoint.Say("Deleting %s...", key)
	if e = registry.DeleteKey(root, path); e != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, e)
	}

	return nil
}

// Kill a process by name
func TaskKillName(process string) bool {
	var cmd []string = []string{"taskkill", "/F", "/IM"}

	if !strings.HasSuffix(process, ".exe") {
		process += ".exe"
	}

	endpoint.Say("Killing %s process...", process)
	if _, e := endpoint.Shell(append(cmd, process)); e != nil {
		endpoint.Say("Failed to kill %s: %s", process, e)
		return false
	}

	endpoint.Say("Successfully killed %s", process)
	return true
}

// Kill a process by PID
func TaskKillPID(pid int) bool {
	var cmd []string = []string{"taskkill", "/F", "/PID"}
	var e error

	endpoint.Say("Killing PID %d...", pid)
	_, e = endpoint.Shell(append(cmd, strconv.Itoa(pid)))
	if e != nil {
		endpoint.Say("Failed to kill PID %d: %s", pid, e)
		return false
	}

	endpoint.Say("Successfully killed PID %d", pid)
	return true
}
