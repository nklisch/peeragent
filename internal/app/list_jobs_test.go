package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nklisch/peeragent/internal/jobs"
)

func TestListJobsSkipsAbsentButReportsCorruptMetadata(t *testing.T) {
	cwd := t.TempDir()
	store := jobs.NewStore(cwd)
	job, err := store.Create(cwd, testJobSpec(), "work")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(job.LogPath), "job.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	service := NewService(Options{})
	records, err := service.ListJobs(context.Background(), cwd, false)
	if err != nil || len(records) != 0 {
		t.Fatalf("missing metadata: %#v %v", records, err)
	}
	if err := os.WriteFile(path, []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListJobs(context.Background(), cwd, false); err == nil || !strings.Contains(err.Error(), "load async job") {
		t.Fatalf("corrupt metadata hidden: %v", err)
	}
}
