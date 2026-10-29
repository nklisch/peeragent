package jobs

import (
	"os"
	"strconv"
	"strings"
)

func bootID() string {
	data, _ := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return strings.TrimSpace(string(data))
}
func processStartTime(pid int) string {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return ""
	}
	return linuxStartTime(string(data))
}
func linuxStartTime(stat string) string {
	end := strings.LastIndexByte(stat, ')')
	if end < 0 {
		return ""
	}
	fields := strings.Fields(stat[end+1:])
	// Fields start at state (3); starttime is field 22.
	if len(fields) < 20 {
		return ""
	}
	return fields[19]
}
