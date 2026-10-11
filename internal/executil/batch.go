package executil

import (
	"fmt"
	"strings"
)

// quoteBatchArg quotes for cmd.exe and for a shim forwarding %* to a native
// argv parser. First apply native argv quoting, then protect metacharacters
// through both cmd parsing passes (invoking the shim and forwarding %*).
// See https://github.com/moxystudio/node-cross-spawn/blob/master/lib/util/escape.js.
func quoteBatchArg(arg string) (string, error) {
	if strings.ContainsAny(arg, "\x00\r\n") {
		return "", fmt.Errorf("Windows batch arguments cannot contain NUL, CR or LF")
	}
	var quoted strings.Builder
	quoted.WriteByte('"')
	backslashes := 0
	for _, char := range arg {
		switch char {
		case '\\':
			backslashes++
		case '"':
			quoted.WriteString(strings.Repeat(`\`, backslashes+1))
			backslashes = 0
		default:
			backslashes = 0
		}
		quoted.WriteRune(char)
	}
	quoted.WriteString(strings.Repeat(`\`, backslashes))
	quoted.WriteByte('"')
	return escapeBatchMeta(escapeBatchMeta(quoted.String())), nil
}

func escapeBatchMeta(value string) string {
	var escaped strings.Builder
	for _, char := range value {
		if strings.ContainsRune("()[]%!^\"`<>&|;, *?\t", char) {
			escaped.WriteByte('^')
		}
		escaped.WriteRune(char)
	}
	return escaped.String()
}
