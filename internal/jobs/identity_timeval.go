package jobs

import (
	"encoding/binary"
	"strconv"
)

// The supported Darwin targets use the same 64-bit timeval prefix. Reading
// only its scalar fields avoids coupling process identity to the full structure.
func darwinStartTime(data []byte, order binary.ByteOrder) string {
	if len(data) < 12 {
		return ""
	}
	seconds, micros := order.Uint64(data[:8]), order.Uint32(data[8:12])
	if seconds == 0 || micros >= 1000000 {
		return ""
	}
	return strconv.FormatUint(seconds, 10) + ":" + strconv.FormatUint(uint64(micros), 10)
}
