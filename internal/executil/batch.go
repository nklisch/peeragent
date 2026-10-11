package executil

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// quoteBatchArg prepares one native argv argument for delayed environment
// expansion at a batch shim's %* forwarding command. cmd.exe does not parse
// the expanded value again, so shell characters and newlines remain data.
func quoteBatchArg(arg string) (string, error) {
	if strings.ContainsRune(arg, '\x00') {
		return "", fmt.Errorf("Windows batch arguments cannot contain NUL")
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
	value := quoted.String()
	// cmd.exe silently drops inherited variables exceeding its string limit.
	// https://learn.microsoft.com/en-us/troubleshoot/windows-client/shell-experience/command-line-string-limitation
	if len(utf16.Encode([]rune(value))) > 8191 {
		return "", fmt.Errorf("Windows batch argument exceeds cmd.exe's 8191-character limit; use a native executable")
	}
	return value, nil
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
