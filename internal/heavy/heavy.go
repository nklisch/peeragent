// Package heavy provides best-effort integration with the optional heavy client.
// A missing client, old client, or unavailable dashboard never fails a job.
package heavy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Step struct {
	Job     string `json:"job"`
	State   string `json:"state"`
	Command string `json:"command"`
}

func run(cwd string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "heavy", args...)
	cmd.Dir = cwd
	return cmd.Output()
}

// Older heavy clients treat unknown subcommands as workloads. Probe help first
// so an optional observation never accidentally submits `ls` or `whoami`.
func supports(cwd, command string) bool {
	data, err := run(cwd, "--help")
	if err != nil {
		return false
	}
	words := strings.FieldsFunc(string(data), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r == '-')
	})
	for _, word := range words {
		if word == command {
			return true
		}
	}
	return false
}

// LaunchEnvironment appends this launcher to the inherited control lineage.
// The target keeps its own owner identity; ancestors gain permission to manage
// its delegated heavy jobs without replacing that identity.
func LaunchEnvironment(env []string, cwd string) []string {
	get := func(key string) string {
		for _, entry := range env {
			name, value, _ := strings.Cut(entry, "=")
			if strings.EqualFold(name, key) {
				return strings.TrimSpace(value)
			}
		}
		return ""
	}
	owner := ""
	if supports(cwd, "whoami") {
		if data, err := run(cwd, "whoami"); err == nil {
			owner = strings.TrimSpace(string(data))
		}
	}
	if owner == "" {
		owner = get("HEAVY_SESSION")
	}
	if owner == "" {
		owner = get("CODEX_THREAD_ID")
	}
	if owner == "" {
		return env
	}
	lineage := get("HEAVY_PARENT_SESSIONS")
	if lineage != "" {
		lineage += ":"
	}
	lineage += owner
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "HEAVY_PARENT_SESSIONS") {
			out = append(out, entry)
		}
	}
	return append(out, "HEAVY_PARENT_SESSIONS="+lineage)
}

type job struct {
	ID      string   `json:"id"`
	Job     string   `json:"job"`
	PID     int      `json:"pid"`
	State   string   `json:"state"`
	Owner   string   `json:"owner"`
	Owners  []string `json:"owners"`
	Command string   `json:"command"`
	Argv    []string `json:"argv"`
}

func decode(data []byte) ([]job, bool) {
	var list []job
	if err := json.Unmarshal(data, &list); err == nil {
		return list, true
	}
	var overview struct {
		Jobs    []job `json:"jobs"`
		Queued  []job `json:"queued"`
		Running []job `json:"running"`
	}
	if err := json.Unmarshal(data, &overview); err != nil {
		return nil, false
	}
	return append(append(overview.Jobs, overview.Queued...), overview.Running...), true
}

func snapshot(cwd string) []job {
	if supports(cwd, "ls") {
		if data, err := run(cwd, "ls", "--json"); err == nil {
			if list, ok := decode(data); ok {
				return list
			}
		}
	}
	url := os.Getenv("HEAVYD_DASHBOARD")
	if url == "" {
		url = "http://127.0.0.1:8473/api/overview"
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil
	}
	list, _ := decode(data)
	return list
}

// Pending matches live process ancestry or the target's recorded session. Never
// match by directory alone: several unrelated jobs can share a worktree.
func Pending(cwd string, rootPID int, session string) []Step {
	list := snapshot(cwd)
	if len(list) == 0 {
		return nil
	}
	processes := processSnapshot()
	var pending []Step
	seen := map[string]bool{}
	for _, item := range list {
		if item.State != "queued" && item.State != "running" {
			continue
		}
		owned := descended(item.PID, rootPID, processes)
		if session != "" {
			owned = owned || item.Owner == session
			for _, owner := range item.Owners {
				owned = owned || owner == session
			}
		}
		if !owned {
			continue
		}
		id := item.ID
		if id == "" {
			id = item.Job
		}
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		command := item.Command
		if command == "" {
			command = strings.Join(item.Argv, " ")
		}
		pending = append(pending, Step{Job: id, State: item.State, Command: command})
	}
	return pending
}

type process struct{ parent, session int }

func descended(pid, root int, processes map[int]process) bool {
	if root <= 1 || pid <= 1 {
		return false
	}
	seen := map[int]bool{}
	for pid > 1 && !seen[pid] {
		if pid == root {
			return true
		}
		seen[pid] = true
		entry, ok := processes[pid]
		if !ok {
			return false
		}
		// Detached workers own a session. Its identity survives a target exiting
		// and orphaning a queued heavy client before handback.
		if entry.session == root {
			return true
		}
		pid = entry.parent
	}
	return false
}
