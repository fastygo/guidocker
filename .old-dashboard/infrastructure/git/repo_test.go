package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepository_Clone(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cmd := exec.Command("git", "init", sourceDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "config", "user.name", "Test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "compose.yml"), []byte("services:"), 0o644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "add", "compose.yml")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "commit", "-m", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	target := filepath.Join(baseDir, "cloned")
	repository := NewGitRepository()
	commit, err := repository.Clone(context.Background(), sourceDir, "", target)
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	if commit == "" {
		t.Fatal("expected commit hash to be returned")
	}
}

func TestRepository_Clone_BranchNotFound(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	cmd := exec.Command("git", "init", sourceDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "config", "user.email", "test@example.com")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "config", "user.name", "Test")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "compose.yml"), []byte("services:"), 0o644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "add", "compose.yml")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	cmd = exec.Command("git", "-C", sourceDir, "commit", "-m", "init")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	target := filepath.Join(baseDir, "cloned-branch")
	repository := NewGitRepository()
	_, err := repository.Clone(context.Background(), sourceDir, "missing-branch", target)
	if err == nil {
		t.Fatal("expected branch not found error")
	}
}
