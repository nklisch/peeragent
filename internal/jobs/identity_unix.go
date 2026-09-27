//go:build unix

package jobs

func ProcessIdentity(int) (uint64, error) { return 0, nil }
