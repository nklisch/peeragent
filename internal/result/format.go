package result

import (
	"encoding/json"
	"fmt"
	"strings"
)

func FormatJSON(res Result) ([]byte, error) {
	return json.Marshal(res)
}

func FormatText(res Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Codex Implement: %s\n\n", res.Status)
	fmt.Fprintf(&b, "Summary:\n%s\n\n", res.Summary)

	if len(res.ChangedFiles) > 0 {
		b.WriteString("Changed Files:\n")
		for _, file := range res.ChangedFiles {
			fmt.Fprintf(&b, "- %s\n", file)
		}
		b.WriteString("\n")
	}

	if len(res.Verification) > 0 {
		b.WriteString("Verification:\n")
		for _, check := range res.Verification {
			fmt.Fprintf(&b, "- %s: %s\n", check.Command, check.Status)
		}
		b.WriteString("\n")
	}

	if res.Details != "" {
		fmt.Fprintf(&b, "Details:\n%s\n\n", res.Details)
	}

	b.WriteString("Metadata:\n")
	if res.Metadata.CWD != "" {
		fmt.Fprintf(&b, "- cwd: %s\n", res.Metadata.CWD)
	}
	if res.Metadata.Access != "" {
		fmt.Fprintf(&b, "- access: %s\n", res.Metadata.Access)
	}
	if res.Metadata.Profile != "" {
		fmt.Fprintf(&b, "- profile: %s\n", res.Metadata.Profile)
	}
	if res.Metadata.Effort != "" {
		fmt.Fprintf(&b, "- effort: %s\n", res.Metadata.Effort)
	}
	fmt.Fprintf(&b, "- exit_code: %d\n", res.Metadata.ExitCode)

	return strings.TrimRight(b.String(), "\n")
}
