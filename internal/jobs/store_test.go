package jobs

import "testing"

func TestCreateAndLoadJob(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("do work")
	if err != nil {
		t.Fatal(err)
	}
	if job.ID == "" {
		t.Fatal("expected id")
	}
	if job.Status != "running" {
		t.Fatalf("Status = %q", job.Status)
	}

	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TaskText != "do work" {
		t.Fatalf("TaskText = %q", loaded.TaskText)
	}
	if loaded.LogPath == "" || loaded.ResultPath == "" {
		t.Fatalf("expected paths: %#v", loaded)
	}
}

func TestSaveJob(t *testing.T) {
	store := NewStore(t.TempDir())
	job, err := store.Create("do work")
	if err != nil {
		t.Fatal(err)
	}
	job.Status = "complete"
	if err := store.Save(job); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "complete" {
		t.Fatalf("Status = %q", loaded.Status)
	}
}
