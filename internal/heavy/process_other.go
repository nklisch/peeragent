//go:build !unix && !windows

package heavy

func processSnapshot() map[int]process { return nil }
