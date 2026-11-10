package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitJobStoreIsPerWorktree(t *testing.T) {
	// Git reports canonical paths: macOS temp folders sit behind a /private link and Windows ones can use 8.3 names.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
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
		job, err := tc.store.Create(tc.cwd, ExecSpec{Agent: "claude"}, "fixture")
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Dir(job.LogPath) != filepath.Join(tc.store.Root, job.ID) {
			t.Fatalf("job path: %#v", job)
		}
		if _, err := os.Stat(filepath.Join(tc.cwd, ".peeragent")); !os.IsNotExist(err) {
			t.Fatalf("Claude job wrote to the checkout: %v", err)
		}
	}

	for _, cwd := range []string{repo, sub} {
		store := NewStore(cwd)
		top := cwd
		if cwd == sub {
			top = worktree
		}
		job, err := store.Create(cwd, ExecSpec{Agent: "codex"}, "workspace-write job")
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(top, ".peeragent", "jobs", job.ID)
		if got := filepath.Dir(job.LogPath); got != want {
			t.Fatalf("Codex job folder = %s, want %s", got, want)
		}
		if loaded, err := NewStore(top).Load(job.ID); err != nil || loaded.CWD != cwd {
			t.Fatalf("checkout-root lookup: %#v, %v", loaded, err)
		}
		if _, err := store.Mailbox(job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(want, "lock.guard")); err != nil {
			t.Fatalf("mailbox guard not in the workspace: %v", err)
		}
		job, err = store.Create(cwd, ExecSpec{Agent: "codex", FullAccess: true}, "full-access job")
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(store.Root, job.ID); filepath.Dir(job.LogPath) != want {
			t.Fatalf("full-access job folder = %s, want %s", filepath.Dir(job.LogPath), want)
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
