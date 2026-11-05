//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// scratch points the registry work at a key of this test's own, and keeps it from
// telling the running system anything. Nothing here touches the real PATH.
func scratch(t *testing.T) userKeys {
	t.Helper()
	base := fmt.Sprintf(`Software\tigwx-test-%d`, os.Getpid())
	old, oldAnnounce := system, announce
	system = userKeys{root: registry.CURRENT_USER, env: base + `\Environment`, app: base + `\Uninstall`}
	announce = func() {}
	t.Cleanup(func() {
		system, announce = old, oldAnnounce
		registry.DeleteKey(registry.CURRENT_USER, base+`\Environment`)
		registry.DeleteKey(registry.CURRENT_USER, base+`\Uninstall`)
		registry.DeleteKey(registry.CURRENT_USER, base)
	})
	k, _, err := registry.CreateKey(registry.CURRENT_USER, system.env, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	k.Close()
	return system
}

func setPath(t *testing.T, u userKeys, value string, expand bool) {
	t.Helper()
	k, err := registry.OpenKey(u.root, u.env, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	if expand {
		err = k.SetExpandStringValue("Path", value)
	} else {
		err = k.SetStringValue("Path", value)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func getPath(t *testing.T, u userKeys) (string, uint32, bool) {
	t.Helper()
	k, err := registry.OpenKey(u.root, u.env, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close()
	v, kind, err := k.GetStringValue("Path")
	if err == registry.ErrNotExist {
		return "", 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return v, kind, true
}

func TestPathStaysTheKindItWas(t *testing.T) {
	u := scratch(t)
	const dir = `C:\t\bin`
	for _, c := range []struct {
		value  string
		expand bool
		kind   uint32
	}{
		{`%USERPROFILE%\go\bin;C:\a`, true, registry.EXPAND_SZ},
		{`C:\a;C:\b`, false, registry.SZ},
	} {
		setPath(t, u, c.value, c.expand)
		if changed, err := u.addToPath(dir); err != nil || !changed {
			t.Fatal(changed, err)
		}
		got, kind, _ := getPath(t, u)
		if got != c.value+";"+dir || kind != c.kind {
			t.Fatalf("%q %d", got, kind)
		}
		if changed, err := u.addToPath(dir); err != nil || changed {
			t.Fatal("twice", changed, err)
		}
		if changed, err := u.removeFromPath(dir); err != nil || !changed {
			t.Fatal(changed, err)
		}
		got, kind, _ = getPath(t, u)
		if got != c.value || kind != c.kind {
			t.Fatalf("%q %d", got, kind)
		}
	}
}

func TestPathThatWasNotThere(t *testing.T) {
	u := scratch(t)
	if _, _, ok := getPath(t, u); ok {
		t.Fatal("the key should be empty")
	}
	if changed, err := u.addToPath(`C:\t\bin`); err != nil || !changed {
		t.Fatal(changed, err)
	}
	got, kind, _ := getPath(t, u)
	if got != `C:\t\bin` || kind != registry.EXPAND_SZ {
		t.Fatal(got, kind)
	}
	if _, err := u.removeFromPath(`C:\t\bin`); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := getPath(t, u); ok {
		t.Fatal("left an empty value")
	}
}

// The whole of an install and an uninstall, against the scratch key.
func TestWholeInstall(t *testing.T) {
	u := scratch(t)
	setPath(t, u, `%USERPROFILE%\go\bin`, true)
	var announced int
	announce = func() { announced++ }
	data := standIn(t)
	self, _ := os.Executable()
	root := filepath.Join(t.TempDir(), "tig wx")
	s, out := portable(root, "")
	s.portable = false

	if code := s.installProgram(data, self, io.Discard); code != exitOK {
		t.Fatalf("%d %s", code, out.String())
	}
	bin := filepath.Join(root, "bin")
	got, _, _ := getPath(t, u)
	if got != `%USERPROFILE%\go\bin;`+bin {
		t.Fatal(got)
	}
	if announced != 1 {
		t.Fatal(announced)
	}
	if _, err := os.Stat(filepath.Join(root, uninstallerName)); err != nil {
		t.Fatal(err)
	}
	k, err := registry.OpenKey(u.root, u.app, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	command, _, _ := k.GetStringValue("UninstallString")
	location, _, _ := k.GetStringValue("InstallLocation")
	name, _, _ := k.GetStringValue("DisplayName")
	k.Close()
	if name != "tigwx" || location != root || !strings.Contains(command, `"`+filepath.Join(root, uninstallerName)+`" -uninstall -dir "`+root+`"`) {
		t.Fatal(name, location, command)
	}

	// An entry for another folder is not this install's to take.
	s2, _ := portable(filepath.Join(t.TempDir(), "elsewhere"), "")
	s2.portable = false
	if code := s2.uninstall(self, io.Discard); code != exitNo {
		t.Fatal(code)
	}
	if k, err := registry.OpenKey(u.root, u.app, registry.QUERY_VALUE); err != nil {
		t.Fatal("took another install's entry")
	} else {
		k.Close()
	}

	s, out = portable(root, "")
	s.portable = false
	if code := s.uninstall(self, io.Discard); code != exitOK {
		t.Fatalf("%d %s", code, out.String())
	}
	got, _, _ = getPath(t, u)
	if got != `%USERPROFILE%\go\bin` {
		t.Fatal(got)
	}
	if _, err := registry.OpenKey(u.root, u.app, registry.QUERY_VALUE); err != registry.ErrNotExist {
		t.Fatal("still listed", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("the folder is still there")
	}
}
