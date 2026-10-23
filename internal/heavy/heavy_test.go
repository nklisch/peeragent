package heavy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PEERAGENT_FAKE_HEAVY") == "1" {
		if path := os.Getenv("FAKE_HEAVY_CALL_LOG"); path != "" {
			file, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			if file != nil {
				fmt.Fprintln(file, strings.Join(os.Args[1:], " "))
				file.Close()
			}
		}
		switch strings.Join(os.Args[1:], " ") {
		case "--help":
			if os.Getenv("FAKE_HEAVY_LEGACY") != "1" {
				fmt.Print("usage: heavy {ls,whoami} ...")
			} else {
				fmt.Print("usage: heavy [options] command ...")
			}
			os.Exit(0)
		case "whoami":
			fmt.Print(os.Getenv("FAKE_HEAVY_OWNER"))
			os.Exit(0)
		case "ls --json":
			fmt.Print(os.Getenv("FAKE_HEAVY_JOBS"))
			os.Exit(0)
		case "receive":
			fmt.Print(os.Getenv("HEAVY_PARENT_SESSIONS"))
			os.Exit(0)
		}
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func fakeHeavy(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "heavy"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	data, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PEERAGENT_FAKE_HEAVY", "1")
	return path
}

func TestLaunchLineageEnvironmentReceived(t *testing.T) {
	path := fakeHeavy(t)
	for _, tc := range []struct{ name, whoami, heavy, codex, inherited, want string }{
		{"whoami", "launcher", "fallback", "thread", "ancestor", "ancestor:launcher"},
		{"heavy session", "", "fallback", "thread", "", "fallback"},
		{"codex thread", "", "", "thread", "ancestor", "ancestor:thread"},
		{"none", "", "", "", "ancestor", "ancestor"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FAKE_HEAVY_OWNER", tc.whoami)
			t.Setenv("HEAVY_SESSION", tc.heavy)
			t.Setenv("CODEX_THREAD_ID", tc.codex)
			t.Setenv("HEAVY_PARENT_SESSIONS", tc.inherited)
			cmd := exec.Command(path, "receive")
			cmd.Env = LaunchEnvironment(os.Environ(), t.TempDir())
			data, err := cmd.Output()
			if err != nil || string(data) != tc.want {
				t.Fatalf("received %q, %v; want %q", data, err, tc.want)
			}
		})
	}
}

func TestMissingHeavyLineageFallback(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HEAVY_SESSION", "session")
	env := LaunchEnvironment([]string{"HEAVY_SESSION=session", "OTHER=kept"}, t.TempDir())
	if !reflect.DeepEqual(env, []string{"HEAVY_SESSION=session", "OTHER=kept", "HEAVY_PARENT_SESSIONS=session"}) {
		t.Fatal(env)
	}
	if got := LaunchEnvironment([]string{"OTHER=kept"}, t.TempDir()); !reflect.DeepEqual(got, []string{"OTHER=kept"}) {
		t.Fatal(got)
	}
}

func TestPendingFakeHeavyAncestryAndOwners(t *testing.T) {
	fakeHeavy(t)
	data := fmt.Sprintf(`{"jobs":[
 {"id":"descendant","pid":%d,"state":"queued","argv":["go","test","./..."]},
 {"id":"owned","state":"running","owners":["ancestor","target"],"command":"cargo build"},
 {"id":"direct-owner","state":"queued","owner":"target","command":"go build"},
 {"id":"finished","state":"succeeded","owner":"target"},
 {"id":"other","state":"running","owner":"someone-else","cwd":%q}
 ]}`, os.Getpid(), t.TempDir())
	t.Setenv("FAKE_HEAVY_JOBS", data)
	got := Pending(t.TempDir(), os.Getppid(), "target")
	want := []Step{{"descendant", "queued", "go test ./..."}, {"owned", "running", "cargo build"}, {"direct-owner", "queued", "go build"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestDashboardFallbackAndAbsentIntegration(t *testing.T) {
	fakeHeavy(t)
	t.Setenv("FAKE_HEAVY_JOBS", "old client: unsupported command")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/overview" {
			t.Errorf("path %q", r.URL.Path)
		}
		fmt.Fprint(w, `{"queued":[{"id":"queue","state":"queued","owner":"session","command":"build"}],"running":[]}`)
	}))
	t.Setenv("HEAVYD_DASHBOARD", server.URL+"/api/overview")
	got := Pending(t.TempDir(), 0, "session")
	if len(got) != 1 || got[0].Job != "queue" {
		t.Fatalf("got %#v", got)
	}
	server.Close()
	t.Setenv("PATH", t.TempDir())
	if got := Pending(t.TempDir(), os.Getpid(), "session"); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestDescendedRejectsUnrelatedAndHandlesOrphans(t *testing.T) {
	processes := map[int]process{10: {parent: 2, session: 10}, 20: {parent: 10, session: 10}, 30: {parent: 1, session: 10}, 40: {parent: 1, session: 40}, 50: {parent: 51}, 51: {parent: 50}}
	for _, tc := range []struct {
		pid, root int
		want      bool
	}{{20, 10, true}, {30, 10, true}, {40, 10, false}, {50, 10, false}, {0, 10, false}, {20, 0, false}} {
		if got := descended(tc.pid, tc.root, processes); got != tc.want {
			t.Fatalf("descended(%d,%d)=%v", tc.pid, tc.root, got)
		}
	}
}

func TestDecodeOverview(t *testing.T) {
	for _, input := range []string{`[{"id":"a"}]`, `{"jobs":[{"id":"a"}]}`, `{"queued":[{"id":"a"}],"running":[]}`} {
		list, ok := decode([]byte(input))
		if !ok || len(list) != 1 || list[0].ID != "a" {
			t.Fatal(input, list, ok)
		}
	}
}

func TestLegacyHeavyNeverSubmitsManagementCommands(t *testing.T) {
	fakeHeavy(t)
	t.Setenv("FAKE_HEAVY_LEGACY", "1")
	t.Setenv("HEAVY_SESSION", "fallback")
	log := filepath.Join(t.TempDir(), "calls")
	t.Setenv("FAKE_HEAVY_CALL_LOG", log)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, `{}`) }))
	defer server.Close()
	t.Setenv("HEAVYD_DASHBOARD", server.URL)
	env := LaunchEnvironment(os.Environ(), t.TempDir())
	if !strings.Contains(strings.Join(env, "\n"), "HEAVY_PARENT_SESSIONS=") {
		t.Fatal("fallback missing")
	}
	Pending(t.TempDir(), 0, "")
	data, err := os.ReadFile(log)
	if err != nil || string(data) != "--help\n--help\n" {
		t.Fatalf("legacy commands: %q %v", data, err)
	}
}
