package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitJobStoreIsPerWorktree(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	git := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "--quiet", repo)
	git("-C", repo, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "-m", "fixture")
	worktree := filepath.Join(root, "worktree")
	git("-C", repo, "worktree", "add", "--quiet", "--detach", worktree)
	primary, linked := NewStore(repo), NewStore(worktree)
	if primary.Root != filepath.Join(repo, ".git", "peeragent", "jobs") || primary.Root == linked.Root {
		t.Fatalf("stores not isolated: %#v %#v", primary, linked)
	}
	sub := filepath.Join(worktree, "nested")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if NewStore(sub).Root != linked.Root {
		t.Fatalf("nested folder did not resolve checkout git dir")
	}
	for _, tc := range []struct {
		cwd   string
		store Store
	}{{repo, primary}, {worktree, linked}} {
		job, err := tc.store.Create(tc.cwd, ExecSpec{}, "fixture")
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Dir(job.LogPath) != filepath.Join(tc.store.Root, job.ID) {
			t.Fatalf("job path: %#v", job)
		}
		if _, err := os.Stat(filepath.Join(tc.cwd, ".peeragent")); !os.IsNotExist(err) {
			t.Fatalf("worktree polluted: %v", err)
		}
	}
}

func TestGitJobStoreDoesNotHideCorruptPrimary(t *testing.T) {
	cwd := t.TempDir()
	out, err := exec.Command("git", "init", "--quiet", cwd).CombinedOutput()
	if err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	store := NewStore(cwd)
	legacy := Store{Root: filepath.Join(cwd, ".peeragent", "jobs")}
	job, err := legacy.Create(cwd, ExecSpec{}, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.Root, job.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "job.json"), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(job.ID); err == nil {
		t.Fatal("corrupt primary job hidden by legacy fallback")
	}
	ids, err := store.ListIDs()
	if err != nil || len(ids) != 1 || ids[0] != job.ID {
		t.Fatalf("duplicate ids: %v %v", ids, err)
	}
}
