package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wwwhana/bookwormwiki/internal/brain"
)

func runWorkspaceTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	gitArgs := append([]string{"-c", "commit.gpgSign=false"}, args...)
	command := exec.Command("git", gitArgs...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestCollectWorkspaceFactsInitializesStableCloneIdentity(t *testing.T) {
	// Ignore url.<base>.insteadOf rewrites injected through GIT_CONFIG_* by
	// the host; the test asserts the remote exactly as it was configured.
	t.Setenv("GIT_CONFIG_COUNT", "0")
	repo := t.TempDir()
	runWorkspaceTestGit(t, repo, "init", "-q")
	runWorkspaceTestGit(t, repo, "config", "user.email", "bookwormwiki-test@example.invalid")
	runWorkspaceTestGit(t, repo, "config", "user.name", "BookwormWiki Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o600); err != nil {
		t.Fatalf("write test repository file: %v", err)
	}
	runWorkspaceTestGit(t, repo, "add", "README.md")
	runWorkspaceTestGit(t, repo, "commit", "-qm", "initial")
	runWorkspaceTestGit(t, repo, "remote", "add", "origin", "git@github.com:example/project.git")

	first, err := collectWorkspaceFacts(context.Background(), repo, "codex", "/projects/example", true)
	if err != nil {
		t.Fatalf("collect first workspace facts: %v", err)
	}
	second, err := collectWorkspaceFacts(context.Background(), repo, "codex", "", true)
	if err != nil {
		t.Fatalf("collect second workspace facts: %v", err)
	}
	if first.RepositoryInstanceID == "" || first.RepositoryInstanceID != second.RepositoryInstanceID {
		t.Fatalf("repository instance ID is not stable: %q, %q", first.RepositoryInstanceID, second.RepositoryInstanceID)
	}
	if second.ProjectNamespace != "/projects/example" {
		t.Fatalf("stored project namespace = %q", second.ProjectNamespace)
	}
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("resolve test repository symlinks: %v", err)
	}
	if first.GitCommonDir != first.GitDir || first.WorktreePath != canonicalRepo {
		t.Fatalf("unexpected main worktree paths: %#v", first)
	}
	if first.Provider != "github" || first.RemoteURL != "git@github.com:example/project.git" {
		t.Fatalf("unexpected remote facts: %#v", first)
	}
}

func TestCollectWorkspaceFactsAllowsRepositoryWithoutFirstCommit(t *testing.T) {
	repo := t.TempDir()
	runWorkspaceTestGit(t, repo, "init", "-q")

	facts, err := collectWorkspaceFacts(context.Background(), repo, "codex", "/projects/empty", true)
	if err != nil {
		t.Fatalf("collect empty repository facts: %v", err)
	}
	if facts.HeadSHA != "" {
		t.Fatalf("empty repository head SHA = %q", facts.HeadSHA)
	}
	if facts.RepositoryInstanceID == "" {
		t.Fatal("empty repository did not receive a stable identity")
	}
}

func TestCollectWorkspaceFactsKeepsLinkedWorktreeIdentityAfterMove(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatalf("create repository directory: %v", err)
	}
	runWorkspaceTestGit(t, repo, "init", "-q")
	runWorkspaceTestGit(t, repo, "config", "user.email", "bookwormwiki-test@example.invalid")
	runWorkspaceTestGit(t, repo, "config", "user.name", "BookwormWiki Test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("test\n"), 0o600); err != nil {
		t.Fatalf("write repository file: %v", err)
	}
	runWorkspaceTestGit(t, repo, "add", "README.md")
	runWorkspaceTestGit(t, repo, "commit", "-qm", "initial")

	linked := filepath.Join(root, "linked")
	runWorkspaceTestGit(t, repo, "worktree", "add", "-qb", "feature", linked)
	before, err := collectWorkspaceFacts(context.Background(), linked, "codex", "/projects/example", true)
	if err != nil {
		t.Fatalf("collect linked worktree facts: %v", err)
	}
	beforeSlot, err := brain.WorkspaceWorktreeSlot(before.GitCommonDir, before.GitDir)
	if err != nil {
		t.Fatalf("linked worktree slot: %v", err)
	}
	beforeKey, err := brain.WorkspaceWorktreeKey(before.RepositoryInstanceID, beforeSlot)
	if err != nil {
		t.Fatalf("linked worktree key: %v", err)
	}

	moved := filepath.Join(root, "moved")
	runWorkspaceTestGit(t, repo, "worktree", "move", linked, moved)
	after, err := collectWorkspaceFacts(context.Background(), moved, "codex", "", true)
	if err != nil {
		t.Fatalf("collect moved worktree facts: %v", err)
	}
	afterSlot, err := brain.WorkspaceWorktreeSlot(after.GitCommonDir, after.GitDir)
	if err != nil {
		t.Fatalf("moved worktree slot: %v", err)
	}
	afterKey, err := brain.WorkspaceWorktreeKey(after.RepositoryInstanceID, afterSlot)
	if err != nil {
		t.Fatalf("moved worktree key: %v", err)
	}
	if beforeKey != afterKey {
		t.Fatalf("linked worktree key changed after git worktree move: %q != %q", beforeKey, afterKey)
	}
	if after.ProjectNamespace != "/projects/example" {
		t.Fatalf("moved worktree lost project namespace: %q", after.ProjectNamespace)
	}
}

func TestWorkspaceFactsSkipsDatabaseBootstrap(t *testing.T) {
	if commandNeedsBootstrap([]string{"bookwormwiki", "workspace", "facts"}) {
		t.Fatal("workspace facts unexpectedly requires database bootstrap")
	}
	if !commandNeedsBootstrap([]string{"bookwormwiki", "worktree", "sync"}) {
		t.Fatal("worktree sync unexpectedly skipped database bootstrap")
	}
}

// A clone set up while the project was called Stash keeps its identity: the
// stash.* keys are read, copied to bookwormwiki.*, and the stored worktree
// key stays the same as before the rename.
func TestCollectWorkspaceFactsKeepsStashEraCloneIdentity(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "0")
	repo := t.TempDir()
	runWorkspaceTestGit(t, repo, "init", "-q")
	runWorkspaceTestGit(t, repo, "remote", "add", "upstream", "https://github.com/example/project.git")
	runWorkspaceTestGit(t, repo, "config", "--local", "stash.repositoryInstanceId", "rii_existing123")
	runWorkspaceTestGit(t, repo, "config", "--local", "stash.projectNamespace", "/projects/demo")
	runWorkspaceTestGit(t, repo, "config", "--local", "stash.remoteName", "upstream")
	runWorkspaceTestGit(t, repo, "config", "--local", "stash.repositoryProviderId", "R_123")

	readOnly, err := collectWorkspaceFacts(context.Background(), repo, "codex", "", false)
	if err != nil {
		t.Fatalf("read-only facts from a Stash-era clone: %v", err)
	}
	if readOnly.RepositoryInstanceID != "rii_existing123" || readOnly.ProjectNamespace != "/projects/demo" || readOnly.RemoteURL != "https://github.com/example/project.git" || readOnly.ProviderRepositoryID != "R_123" {
		t.Fatalf("legacy identity not used: %#v", readOnly)
	}
	if out, _ := exec.Command("git", "-C", repo, "config", "--local", "--get", "bookwormwiki.repositoryInstanceId").Output(); len(out) != 0 {
		t.Fatalf("read-only run wrote the new key: %q", out)
	}
	migrated, err := collectWorkspaceFacts(context.Background(), repo, "codex", "", true)
	if err != nil {
		t.Fatalf("initializing facts from a Stash-era clone: %v", err)
	}
	if migrated.RepositoryInstanceID != "rii_existing123" {
		t.Fatalf("instance ID changed: %q", migrated.RepositoryInstanceID)
	}
	for key, want := range map[string]string{"repositoryInstanceId": "rii_existing123", "projectNamespace": "/projects/demo", "remoteName": "upstream", "repositoryProviderId": "R_123"} {
		out, err := exec.Command("git", "-C", repo, "config", "--local", "--get", "bookwormwiki."+key).Output()
		if err != nil || string(out) != want+"\n" {
			t.Fatalf("bookwormwiki.%s = %q, %v", key, out, err)
		}
	}
	// Keys stored before the rename must still match: the salt is frozen.
	sum := sha256.Sum256([]byte("stash-worktree-v1\x00rii_existing123\x00main"))
	if got, err := brain.WorkspaceWorktreeKey("rii_existing123", "main"); err != nil || got != "wtk_v1_"+hex.EncodeToString(sum[:]) {
		t.Fatalf("worktree key = %q, %v", got, err)
	}
}
