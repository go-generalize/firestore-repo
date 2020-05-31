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
		tr.Parallel()
		if err := os.Chdir(filepath.Join(root, "testfiles/auto")); err != nil {
			tr.Fatalf("chdir failed: %+v", err)
		}

		if err := run("Task"); err != nil {
			tr.Fatalf("failed to generate for testfiles/a: %+v", err)
		}

		execTest(tr)
	})

	t.Run("IDSpecified", func(tr *testing.T) {
		tr.Parallel()
		if err := os.Chdir(filepath.Join(root, "testfiles/auto")); err != nil {
			tr.Fatalf("chdir failed: %+v", err)
		}

		if err := run("Task"); err != nil {
			tr.Fatalf("failed to generate for testfiles/b: %+v", err)
		}

		execTest(tr)
	})
}
