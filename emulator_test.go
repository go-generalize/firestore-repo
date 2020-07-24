// +build emulator

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func execTest(t *testing.T) {
	t.Helper()

	b, err := exec.Command("go", "test", "./tests", "-v", "-tags", "internal").CombinedOutput()

	if err != nil {
		t.Fatalf("go test failed: %+v(%s)", err, string(b))
	}
}

func TestGenerator(t *testing.T) {
	root, err := os.Getwd()

	if err != nil {
		t.Fatalf("failed to getwd: %+v", err)
	}

	t.Run("AutomaticIDGeneration", func(tr *testing.T) {
		if err := os.Chdir(filepath.Join(root, "testfiles/auto")); err != nil {
			tr.Fatalf("chdir failed: %+v", err)
		}

		if err := run("Task", true, false); err != nil {
			tr.Fatalf("failed to generate for testfiles/auto: %+v", err)
		}

		if err := run("Lock", false, false); err != nil {
			tr.Fatalf("failed to generate for testfiles/auto: %+v", err)
		}

		execTest(tr)
	})

	t.Run("IDSpecified", func(tr *testing.T) {
		if err := os.Chdir(filepath.Join(root, "testfiles/not_auto")); err != nil {
			tr.Fatalf("chdir failed: %+v", err)
		}

		if err := run("Task", true, false); err != nil {
			tr.Fatalf("failed to generate for testfiles/not_auto: %+v", err)
		}

		if err := run("SubTask", true, true); err != nil {
			tr.Fatalf("failed to generate for testfiles/not_auto: %+v", err)
		}

		execTest(tr)
	})
}
