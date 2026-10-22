package jobs

import (
	"os"
	"strconv"
	"strings"
)

// A zombie still answers kill(pid, 0), but cannot write the agent session.
// procfs failure leaves the existing signal probe in place; it is not a refusal.
func processExited(pid int) bool {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	// comm is parenthesized and may itself contain spaces and parentheses.
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 || len(data) <= end+2 {
		return false
	}
	return data[end+2] == 'Z' || data[end+2] == 'X'
}
