package heavy

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func processSnapshot() map[int]process {
	out := map[int]process{}
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		// comm is parenthesized and may itself contain spaces or parentheses.
		end := strings.LastIndexByte(string(data), ')')
		if end < 0 {
			continue
		}
		fields := strings.Fields(string(data[end+1:]))
		if len(fields) < 4 {
			continue
		}
		parent, _ := strconv.Atoi(fields[1])
		session, _ := strconv.Atoi(fields[3])
		out[pid] = process{parent: parent, session: session}
	}
	return out
}
