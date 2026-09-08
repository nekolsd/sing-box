//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestInstalledApplicationPath(t *testing.T) {
	installationDirectory := t.TempDir()
	applicationPath := filepath.Join(installationDirectory, "sing-box-nekolsd.exe")
	err := os.WriteFile(applicationPath, nil, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	resolvedDirectory, resolvedApplication, err := installedApplicationPath(
		filepath.Join(installationDirectory, "resources", "daemon", "sing-box-daemon.exe"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedDirectory != installationDirectory || resolvedApplication != applicationPath {
		t.Fatalf("unexpected installed application: %q in %q", resolvedApplication, resolvedDirectory)
	}
	application, err := openLockedExecutable(resolvedApplication)
	if err != nil {
		t.Fatalf("cannot open the packaged application: %v", err)
	}
	windows.CloseHandle(application)

	for _, daemonPath := range []string{
		filepath.Join(installationDirectory, "sing-box-daemon.exe"),
		filepath.Join(installationDirectory, "resources", "other", "sing-box-daemon.exe"),
		filepath.Join(installationDirectory, "resources", "daemon", "other.exe"),
	} {
		if _, _, err = installedApplicationPath(daemonPath); err == nil {
			t.Errorf("accepted invalid daemon layout: %q", daemonPath)
		}
	}
}
