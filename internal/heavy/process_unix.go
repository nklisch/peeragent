//go:build unix && !linux

package heavy

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func processSnapshot() map[int]process {
	out := map[int]process{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		parent, _ := strconv.Atoi(fields[1])
		out[pid] = process{parent: parent}
	}
	return out
}
