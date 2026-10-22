package input

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nklisch/peeragent/internal/agent"
	"github.com/nklisch/peeragent/internal/jobs"
)

const ValidOptions = "--agent --async --sandbox --full-access --worktree --effort --model --profile --resume --cwd --prompt-file --json --text --status --result --wait --wait-any --cancel --retry-capacity --capacity-backoff --help -h"

type JobReference struct {
	JobID string
	CWD   string
}

type Request struct {
	StatusJobs      []JobReference
	WaitAnyJobs     []JobReference
	RetryCapacity   int
	CapacityBackoff time.Duration
	TaskText        string
	CWD             string
	JSON            bool
	Agent           string
	FullAccess      bool
	Worktree        bool
	Profile         string
	Effort          string
	Model           string
	Resume          string
	Async           bool
	JobRunID        string
	StatusJobID     string
	ResultJobID     string
	WaitJobID       string
	CancelJobID     string
	Help            bool
}

func Parse(args []string, stdin io.Reader, getwd func() (string, error)) (Request, error) {
	parsed, err := parseArgs(args)
	if err != nil {
		return Request{}, err
	}

	if parsed.help {
		return Request{Help: true, JSON: parsed.json}, nil
	}
	if err := normalizeJobIDs(&parsed); err != nil {
		return Request{}, err
	}

	if parsed.cwd == "" && len(parsed.statusIDs) <= 1 && len(parsed.waitAnyIDs) == 0 {
		id := ""
		for _, candidate := range []string{parsed.jobRunID, parsed.statusJobID, parsed.resultJobID, parsed.waitJobID, parsed.cancelJobID} {
			if candidate != "" {
				id = candidate
				break
			}
		}
		var cwd string
		var lookupErr error
		if id != "" {
			cwd, lookupErr = jobs.LookupJob(id)
		} else if parsed.resume != "" {
			cwd, lookupErr = jobs.LookupSession(parsed.agent, parsed.resume)
		}
		if lookupErr != nil {
			return Request{}, fmt.Errorf("lookup working directory (use --cwd to override): %w", lookupErr)
		}
		parsed.cwd = cwd
	}
	parts := make([]string, 0, 3)
	if joinedArgs := strings.TrimSpace(strings.Join(parsed.positionals, " ")); joinedArgs != "" {
		parts = append(parts, joinedArgs)
	}

	if parsed.promptFile != "" {
		content, err := os.ReadFile(parsed.promptFile)
		if err != nil {
			return Request{}, fmt.Errorf("read prompt file: %w", err)
		}
		if text := strings.TrimSpace(string(content)); text != "" {
			parts = append(parts, text)
		}
	}

	if shouldReadStdin(stdin, parsed) {
		content, err := io.ReadAll(stdin)
		if err != nil {
			return Request{}, fmt.Errorf("read stdin: %w", err)
		}
		if text := strings.TrimSpace(string(content)); text != "" {
			parts = append(parts, text)
		}
	}

	taskText := strings.Join(parts, "\n\n")
	controlRequest := len(parsed.statusIDs) > 0 || len(parsed.waitAnyIDs) > 0 || parsed.statusJobID != "" || parsed.resultJobID != "" ||
		parsed.waitJobID != "" || parsed.cancelJobID != "" || parsed.jobRunID != ""
	delegation := Delegation{
		TaskText:   taskText,
		CWD:        parsed.cwd,
		Agent:      parsed.agent,
		FullAccess: parsed.fullAccess,
		Profile:    parsed.profile,
		Effort:     parsed.effort,
		Model:      parsed.model,
		Resume:     parsed.resume,
	}
	if controlRequest && strings.TrimSpace(delegation.TaskText) == "" {
		// Control operations have no delegation task, but still pass their
		// target options through the canonical validator so malformed options
		// cannot be accepted only because no target will run in this process.
		delegation.TaskText = "control operation"
	}
	delegation, err = NormalizeDelegation(delegation, getwd)
	if err != nil {
		return Request{}, err
	}
	if controlRequest {
		delegation.TaskText = ""
	}

	return Request{
		StatusJobs:      parsed.statusJobs,
		WaitAnyJobs:     parsed.waitAnyJobs,
		RetryCapacity:   parsed.retryCapacity,
		CapacityBackoff: parsed.capacityBackoff,
		TaskText:        delegation.TaskText,
		CWD:             delegation.CWD,
		JSON:            parsed.json,
		Agent:           delegation.Agent,
		FullAccess:      delegation.FullAccess,
		Worktree:        parsed.worktree,
		Profile:         delegation.Profile,
		Effort:          delegation.Effort,
		Model:           delegation.Model,
		Resume:          delegation.Resume,
		Async:           parsed.async,
		JobRunID:        parsed.jobRunID,
		StatusJobID:     parsed.statusJobID,
		ResultJobID:     parsed.resultJobID,
		WaitJobID:       parsed.waitJobID,
		CancelJobID:     parsed.cancelJobID,
	}, nil
}

func normalizeJobIDs(parsed *parsedArgs) error {
	for _, group := range []struct {
		ids  []string
		refs *[]JobReference
	}{
		{parsed.statusIDs, &parsed.statusJobs}, {parsed.waitAnyIDs, &parsed.waitAnyJobs},
	} {
		for _, raw := range group.ids {
			id, cwd, explicit := strings.Cut(raw, "@")
			id = strings.TrimSpace(id)
			if err := jobs.ValidateID(id); err != nil {
				return err
			}
			if explicit && strings.TrimSpace(cwd) == "" {
				return errors.New("job-id@directory requires a non-empty directory")
			}
			if !explicit {
				cwd = parsed.cwd
			}
			if cwd != "" {
				absolute, err := filepath.Abs(cwd)
				if err != nil {
					return err
				}
				cwd = absolute
			}
			*group.refs = append(*group.refs, JobReference{JobID: id, CWD: cwd})
		}
	}
	if len(parsed.statusJobs) == 1 {
		parsed.statusJobID = parsed.statusJobs[0].JobID
		if parsed.statusJobs[0].CWD != "" {
			parsed.cwd = parsed.statusJobs[0].CWD
		}
	}

	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "--job-run", value: &parsed.jobRunID},
		{name: "--status", value: &parsed.statusJobID},
		{name: "--result", value: &parsed.resultJobID},
		{name: "--wait", value: &parsed.waitJobID},
		{name: "--cancel", value: &parsed.cancelJobID},
	} {
		if *field.value == "" {
			continue
		}
		id := strings.TrimSpace(*field.value)
		if id == "" {
			return fmt.Errorf("%s requires a valid job id", field.name)
		}
		if err := jobs.ValidateID(id); err != nil {
			return fmt.Errorf("%s: %w", field.name, err)
		}
		*field.value = id
	}
	return nil
}

func shouldReadStdin(stdin io.Reader, parsed parsedArgs) bool {
	if stdin == nil || isInteractiveTTY(stdin) || len(parsed.statusIDs) > 0 || len(parsed.waitAnyIDs) > 0 {
		return false
	}
	if _, ok := stdin.(*os.File); !ok {
		return true
	}
	return len(parsed.positionals) == 0 &&
		parsed.promptFile == "" &&
		parsed.jobRunID == "" &&
		parsed.statusJobID == "" &&
		parsed.resultJobID == "" &&
		parsed.waitJobID == "" &&
		parsed.cancelJobID == ""
}

func isInteractiveTTY(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

type parsedArgs struct {
	statusIDs, waitAnyIDs   []string
	statusJobs, waitAnyJobs []JobReference
	retryCapacity           int
	capacityBackoff         time.Duration
	backoffSet              bool
	retrySet                bool
	cwd                     string
	promptFile              string
	json                    bool
	agent                   string
	sandbox                 bool
	fullAccess              bool
	worktree                bool
	profile                 string
	effort                  string
	model                   string
	resume                  string
	async                   bool
	jobRunID                string
	statusJobID             string
	resultJobID             string
	waitJobID               string
	cancelJobID             string
	help                    bool
	positionals             []string
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{json: true, agent: string(agent.DefaultID()), capacityBackoff: 10 * time.Minute}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			parsed.positionals = append(parsed.positionals, args[i+1:]...)
			break
		}
		switch arg {
		case "--cwd", "--prompt-file", "--agent", "--profile", "--effort", "--model", "--resume", "--job-run", "--status", "--result", "--wait", "--wait-any", "--cancel", "--retry-capacity", "--capacity-backoff":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" || strings.HasPrefix(args[i+1], "-") {
				return parsedArgs{}, fmt.Errorf("%s requires a non-empty value", arg)
			}
		}
		switch arg {
		case "--retry-capacity":
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				return parsedArgs{}, errors.New("--retry-capacity requires a non-negative integer")
			}
			parsed.retryCapacity = n
			parsed.retrySet = true
		case "--capacity-backoff":
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil || d <= 0 {
				return parsedArgs{}, errors.New("--capacity-backoff requires a positive duration, e.g. 10m")
			}
			parsed.capacityBackoff = d
			parsed.backoffSet = true
		case "--cwd":
			i++
			parsed.cwd = args[i]
		case "--prompt-file":
			i++
			parsed.promptFile = args[i]
		case "--json":
			parsed.json = true
		case "--text":
			parsed.json = false
		case "--agent":
			i++
			agent, err := normalizeAgent(args[i])
			if err != nil {
				return parsedArgs{}, err
			}
			parsed.agent = agent
		case "--sandbox":
			parsed.sandbox = true
		case "--full-access":
			parsed.fullAccess = true
		case "--worktree":
			parsed.worktree = true
		case "--profile":
			i++
			parsed.profile = args[i]
		case "--effort":
			i++
			parsed.effort = args[i]
		case "--model":
			i++
			parsed.model = args[i]
		case "--resume":
			i++
			parsed.resume = strings.TrimSpace(args[i])
		case "--async":
			parsed.async = true
		case "--job-run":
			i++
			parsed.jobRunID = args[i]
		case "--status", "--wait-any":
			var ids []string
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Single --status historically ignores trailing task text. Only
				// consume additional job references, retaining that behavior.
				if arg == "--status" && len(ids) > 0 {
					id, _, _ := strings.Cut(args[i+1], "@")
					if jobs.ValidateID(strings.TrimSpace(id)) != nil {
						break
					}
				}
				i++
				ids = append(ids, args[i])
			}
			if arg == "--status" {
				parsed.statusIDs = append(parsed.statusIDs, ids...)
			} else {
				parsed.waitAnyIDs = append(parsed.waitAnyIDs, ids...)
			}
		case "--result":
			i++
			parsed.resultJobID = args[i]
		case "--wait":
			i++
			parsed.waitJobID = args[i]
		case "--cancel":
			i++
			parsed.cancelJobID = args[i]
		case "--help", "-h":
			parsed.help = true
		default:
			if strings.HasPrefix(arg, "-") {
				return parsedArgs{}, fmt.Errorf("unknown option %q; valid options: %s; use -- before task text beginning with a dash", arg, ValidOptions)
			}
			parsed.positionals = append(parsed.positionals, arg)
		}
	}
	if parsed.help {
		return parsed, nil
	}
	if len(parsed.waitAnyIDs) > 0 || len(parsed.statusIDs) > 0 {
		controls := 0
		for _, set := range []bool{len(parsed.waitAnyIDs) > 0, len(parsed.statusIDs) > 0, parsed.waitJobID != "", parsed.resultJobID != "", parsed.cancelJobID != "", parsed.jobRunID != "", parsed.async} {
			if set {
				controls++
			}
		}
		if controls > 1 {
			return parsedArgs{}, errors.New("job controls cannot be combined")
		}
	}
	if (parsed.retrySet || parsed.backoffSet) && (!parsed.async || parsed.agent != "codex" || parsed.jobRunID != "" || len(parsed.statusIDs) > 0 || len(parsed.waitAnyIDs) > 0 || parsed.statusJobID != "" || parsed.waitJobID != "" || parsed.resultJobID != "" || parsed.cancelJobID != "") {
		return parsedArgs{}, errors.New("capacity retry options require --async --agent codex")
	}
	if parsed.sandbox && parsed.fullAccess {
		return parsedArgs{}, errors.New("--sandbox cannot be combined with --full-access")
	}
	if parsed.sandbox && parsed.worktree {
		return parsedArgs{}, errors.New("--sandbox cannot be combined with --worktree")
	}
	return parsed, nil
}

func normalizeAgent(raw string) (string, error) {
	canonical, ok := agent.Normalize(raw)
	if !ok {
		return "", errors.New("--agent must be codex, gemini, claude, or zai")
	}
	return string(canonical), nil
}

func normalizeEffort(agent string, effort string) (string, error) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	switch agent {
	case "codex":
		if effort == "" {
			return "high", nil
		}
		switch effort {
		case "low", "medium", "high", "xhigh":
			return effort, nil
		default:
			return "", errors.New("--effort for codex must be low, medium, high, or xhigh")
		}
	case "claude":
		if effort == "" {
			return "xhigh", nil
		}
		switch effort {
		case "high", "xhigh":
			return effort, nil
		default:
			return "", errors.New("--effort for claude must be high or xhigh")
		}
	case "zai":
		if effort == "" {
			return "high", nil
		}
		switch effort {
		case "medium", "high", "xhigh":
			return effort, nil
		default:
			return "", errors.New("--effort for zai must be medium, high, or xhigh")
		}
	case "gemini":
		if effort == "" {
			return "high", nil
		}
		switch effort {
		case "low", "medium", "high":
			return effort, nil
		default:
			return "", errors.New("--effort for gemini must be low, medium, or high")
		}
	default:
		return "", errors.New("--effort is supported only for codex, gemini, claude, or zai")
	}
}

func normalizeModel(agent string, model string) (string, error) {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		switch agent {
		case "gemini":
			return "gemini-3.7-flash", nil
		case "zai":
			return "glm-5.2", nil
		default:
			return "", nil
		}
	}
	switch agent {
	case "codex":
		switch model {
		case "astra", "gpt6-astra", "gpt-6-astra":
			return "gpt-6-astra", nil
		case "luna", "gpt6-luna", "gpt-6-luna":
			return "gpt-6-luna", nil
		case "gpt5.6-luna", "gpt-5.6-luna":
			return "gpt-5.6-luna", nil
		case "terra", "gpt5.6-terra", "gpt-5.6-terra":
			return "gpt-5.6-terra", nil
		case "sol", "gpt6.1-sol", "gpt-6.1-sol":
			return "gpt-6.1-sol", nil
		case "gpt6-sol", "gpt-6-sol":
			return "gpt-6-sol", nil
		case "gpt5.6-sol", "gpt-5.6-sol":
			return "gpt-5.6-sol", nil
		default:
			return "", errors.New("--model for codex must be astra, luna, terra, sol, gpt-6.1-sol, or a supported gpt-6-<astra|luna|sol> or gpt-5.6-<luna|terra|sol> model id")
		}
	case "claude":
		switch model {
		case "fable", "sonnet", "opus", "haiku":
			return model, nil
		default:
			return "", errors.New("--model for claude must be fable, sonnet, opus, or haiku")
		}
	case "gemini":
		switch model {
		case "gemini", "flash", "gemini-3.7", "3.7", "gemini-3.7-flash":
			return "gemini-3.7-flash", nil
		case "gemini-3.6", "3.6", "gemini-3.6-flash":
			return "gemini-3.6-flash", nil
		case "gemini-3.5", "3.5", "gemini-3.5-flash":
			return "gemini-3.5-flash", nil
		case "pro", "gemini-pro", "gemini-3.1", "3.1", "gemini-3.1-pro":
			return "gemini-3.1-pro", nil
		default:
			return "", errors.New("--model for gemini must be flash, pro, or a supported gemini-<version>-<family> id")
		}
	case "zai":
		switch model {
		case "glm", "glm-5.2", "glm5.2", "zai/glm-5.2", "z.ai/glm-5.2":
			return "glm-5.2", nil
		default:
			return "", errors.New("--model for zai is fixed to glm-5.2")
		}
	default:
		return "", errors.New("--model is supported only for codex, claude, gemini, or zai")
	}
}
