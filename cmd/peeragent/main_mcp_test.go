package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestMCPSubprocessKeepsStdoutProtocolPure(t *testing.T) {
	cmd := exec.Command(os.Args[0], "mcp")
	cmd.Env = append(os.Environ(), "PEERAGENT_TEST_HELPER_MAIN=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(stdoutPipe)
	writeFrame := func(frame string) {
		t.Helper()
		if _, err := fmt.Fprintln(stdin, frame); err != nil {
			t.Fatal(err)
		}
	}
	readFrame := func() string {
		t.Helper()
		frames := make(chan string, 1)
		errs := make(chan error, 1)
		go func() {
			line, err := reader.ReadString('\n')
			if err != nil {
				errs <- err
				return
			}
			frames <- line
		}()
		select {
		case line := <-frames:
			return line
		case err := <-errs:
			t.Fatalf("read protocol frame: %v\nstderr:\n%s", err, stderr.String())
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for protocol frame\nstderr:\n%s", stderr.String())
		}
		return ""
	}

	writeFrame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke-client","version":"1.0.0"}}}`)
	initialize := readFrame()
	writeFrame(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`)
	writeFrame(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	tools := readFrame()
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("MCP subprocess failed: %v\nstdout frames:\n%s%s\nstderr:\n%s", err, initialize, tools, stderr.String())
	}

	for _, line := range []string{initialize, tools} {
		var frame map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &frame); err != nil {
			t.Fatalf("stdout contained non-protocol line %q: %v", line, err)
		}
		if frame["jsonrpc"] != "2.0" {
			t.Fatalf("stdout frame missing jsonrpc marker: %#v", frame)
		}
	}
}
