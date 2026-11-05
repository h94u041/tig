//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// userKeys says where in the registry this works, so that a test can point it
// somewhere of its own.
type userKeys struct {
	root registry.Key
	env  string // holds the Path value of this user
	app  string // the entry in the list of installed programs
}

var system = userKeys{
	root: registry.CURRENT_USER,
	env:  `Environment`,
	app:  `Software\Microsoft\Windows\CurrentVersion\Uninstall\tigwx`,
}

// announce tells running programs that the environment changed.
var announce = func() {
	send := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
	name, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001A
		abortIfHung     = 0x0002
	)
	var result uintptr
	send.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(name)), abortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}

func defaultRoot() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		if base, err = os.UserCacheDir(); err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "Programs", "tigwx"), nil
}

func registerPlan(bin string) []string {
	return []string{"adds " + bin + " to your PATH", "adds tigwx to your list of installed programs"}
}

func unregisterPlan(bin string) []string {
	return []string{"takes " + bin + " off your PATH", "takes tigwx off your list of installed programs"}
}

func register(root, bin, program string, size int64, version string) error {
	changed, err := system.addToPath(bin)
	if err != nil {
		return err
	}
	if err := system.listApp(root, program, size, version); err != nil {
		return err
	}
	if changed {
		announce()
	}
	return nil
}

func unregister(root, bin string) error {
	changed, err := system.removeFromPath(bin)
	if err != nil {
		return err
	}
	if err := system.unlistApp(root); err != nil {
		return err
	}
	if changed {
		announce()
	}
	return nil
}

// readPath reads the Path value and how it is stored. A Path holding %VAR% has
// to stay the kind that expands them.
func (u userKeys) readPath(k registry.Key) (string, uint32, error) {
	value, kind, err := k.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", registry.EXPAND_SZ, nil
	}
	if err != nil {
		return "", 0, fmt.Errorf("reading your PATH: %w", err)
	}
	return value, kind, nil
}

func (u userKeys) writePath(k registry.Key, value string, kind uint32) error {
	var err error
	switch {
	case value == "":
		err = k.DeleteValue("Path")
	case kind == registry.EXPAND_SZ:
		err = k.SetExpandStringValue("Path", value)
	default:
		err = k.SetStringValue("Path", value)
	}
	if err != nil {
		return fmt.Errorf("writing your PATH: %w", err)
	}
	return nil
}

func (u userKeys) addToPath(dir string) (bool, error) {
	k, _, err := registry.CreateKey(u.root, u.env, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	value, kind, err := u.readPath(k)
	if err != nil {
		return false, err
	}
	next, changed := pathWith(value, dir)
	if !changed {
		return false, nil
	}
	return true, u.writePath(k, next, kind)
}

func (u userKeys) removeFromPath(dir string) (bool, error) {
	k, err := registry.OpenKey(u.root, u.env, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer k.Close()
	value, kind, err := u.readPath(k)
	if err != nil {
		return false, err
	}
	next, changed := pathWithout(value, dir)
	if !changed {
		return false, nil
	}
	return true, u.writePath(k, next, kind)
}

func (u userKeys) listApp(root, program string, size int64, version string) error {
	k, _, err := registry.CreateKey(u.root, u.app, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	uninstall := fmt.Sprintf(`"%s" -uninstall -dir "%s"`, filepath.Join(root, uninstallerName), root)
	for name, value := range map[string]string{
		"DisplayName":          "tigwx",
		"DisplayVersion":       version,
		"Publisher":            "tig",
		"InstallLocation":      root,
		"DisplayIcon":          program,
		"URLInfoAbout":         "https://github.com/h94u041/tig",
		"UninstallString":      uninstall,
		"QuietUninstallString": uninstall + " -yes",
	} {
		if err := k.SetStringValue(name, value); err != nil {
			return err
		}
	}
	for name, value := range map[string]uint32{"NoModify": 1, "NoRepair": 1, "EstimatedSize": uint32(size / 1024)} {
		if err := k.SetDWordValue(name, value); err != nil {
			return err
		}
	}
	return nil
}

// unlistApp takes the entry away if it is for this folder: another install
// keeps its own.
func (u userKeys) unlistApp(root string) error {
	k, err := registry.OpenKey(u.root, u.app, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	location, _, err := k.GetStringValue("InstallLocation")
	k.Close()
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	if !sameFolder(location, root) {
		return nil
	}
	return registry.DeleteKey(u.root, u.app)
}

// removeAfterExit is what a running program on Windows has for removing its
// own file: something else does it once this has ended. Only the file, and the
// folders if nothing else is in them.
func removeAfterExit(self, root string) error {
	if strings.ContainsAny(self+root, `"%`) {
		return errors.New("the path has characters this cannot pass on; delete it by hand")
	}
	bin := filepath.Join(root, "bin")
	line := fmt.Sprintf(`cmd.exe /d /s /c "ping -n 3 127.0.0.1 >nul & del /f /q "%s" & rmdir "%s" & rmdir "%s""`, self, bin, root)
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       line,
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
	cmd.Dir = os.TempDir()
	return cmd.Start()
}
