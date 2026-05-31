package input

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Request struct {
	TaskText    string
	CWD         string
	JSON        bool
	Agent       string
	FullAccess  bool
	Worktree    bool
	Profile     string
	Effort      string
	Model       string
	Resume      string
	Async       bool
	JobRunID    string
	StatusJobID string
	ResultJobID string
	CancelJobID string
	Help        bool
}

func Parse(args []string, stdin io.Reader, getwd func() (string, error)) (Request, error) {
	parsed, err := parseArgs(args)
	if err != nil {
		return Request{}, err
	}

	if parsed.help {
		return Request{Help: true, JSON: parsed.json}, nil
	}

	cwd := strings.TrimSpace(parsed.cwd)
	if cwd == "" {
		var err error
		cwd, err = getwd()
		if err != nil {
			return Request{}, fmt.Errorf("resolve cwd: %w", err)
		}
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
	if taskText == "" {
		if parsed.statusJobID != "" || parsed.resultJobID != "" ||
			parsed.cancelJobID != "" || parsed.jobRunID != "" {
			taskText = ""
		} else {
			return Request{}, errors.New("no task text supplied")
		}
	}

	return Request{
		TaskText:    taskText,
		CWD:         cwd,
		JSON:        parsed.json,
		Agent:       parsed.agent,
		FullAccess:  parsed.fullAccess,
		Worktree:    parsed.worktree,
		Profile:     parsed.profile,
		Effort:      parsed.effort,
		Model:       parsed.model,
		Resume:      parsed.resume,
		Async:       parsed.async,
		JobRunID:    parsed.jobRunID,
		StatusJobID: parsed.statusJobID,
		ResultJobID: parsed.resultJobID,
		CancelJobID: parsed.cancelJobID,
	}, nil
}

func shouldReadStdin(stdin io.Reader, parsed parsedArgs) bool {
	if stdin == nil || isInteractiveTTY(stdin) {
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
	cwd         string
	promptFile  string
	json        bool
	agent       string
	sandbox     bool
	fullAccess  bool
	worktree    bool
	profile     string
	effort      string
	model       string
	resume      string
	async       bool
	jobRunID    string
	statusJobID string
	resultJobID string
	cancelJobID string
	help        bool
	positionals []string
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{json: true, agent: "codex"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--cwd":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--cwd requires a value")
			}
			parsed.cwd = args[i]
		case "--prompt-file":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--prompt-file requires a value")
			}
			parsed.promptFile = args[i]
		case "--json":
			parsed.json = true
		case "--text":
			parsed.json = false
		case "--agent":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--agent requires a value")
			}
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
			if i >= len(args) {
				return parsedArgs{}, errors.New("--profile requires a value")
			}
			parsed.profile = args[i]
		case "--effort":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--effort requires a value")
			}
			parsed.effort = args[i]
		case "--model":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--model requires a value")
			}
			parsed.model = args[i]
		case "--resume":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--resume requires a session id")
			}
			parsed.resume = strings.TrimSpace(args[i])
		case "--async":
			parsed.async = true
		case "--job-run":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--job-run requires a value")
			}
			parsed.jobRunID = args[i]
		case "--status":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--status requires a job id")
			}
			parsed.statusJobID = args[i]
		case "--result":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--result requires a job id")
			}
			parsed.resultJobID = args[i]
		case "--cancel":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--cancel requires a job id")
			}
			parsed.cancelJobID = args[i]
		case "--help", "-h":
			parsed.help = true
		default:
			parsed.positionals = append(parsed.positionals, arg)
		}
	}
	if parsed.help {
		return parsed, nil
	}
	if parsed.sandbox && parsed.fullAccess {
		return parsedArgs{}, errors.New("--sandbox cannot be combined with --full-access")
	}
	if parsed.sandbox && parsed.worktree {
		return parsedArgs{}, errors.New("--sandbox cannot be combined with --worktree")
	}
	effort, err := normalizeEffort(parsed.agent, parsed.effort)
	if err != nil {
		return parsedArgs{}, err
	}
	parsed.effort = effort
	model, err := normalizeModel(parsed.agent, parsed.model)
	if err != nil {
		return parsedArgs{}, err
	}
	parsed.model = model
	return parsed, nil
}

func normalizeAgent(agent string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "", "codex":
		return "codex", nil
	case "gemini", "agy", "antigravity":
		return "gemini", nil
	case "claude":
		return "claude", nil
	default:
		return "", errors.New("--agent must be codex, gemini, or claude")
	}
}

func normalizeEffort(agent string, effort string) (string, error) {
	effort = strings.ToLower(strings.TrimSpace(effort))
	switch agent {
	case "codex":
		if effort == "" {
			return "high", nil
		}
		switch effort {
		case "medium", "high", "xhigh":
			return effort, nil
		default:
			return "", errors.New("--effort for codex must be medium, high, or xhigh")
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
	case "gemini":
		if effort == "" {
			return "", nil
		}
		return "", errors.New("--effort is supported only for codex or claude")
	default:
		return "", errors.New("--effort is supported only for codex or claude")
	}
}

func normalizeModel(agent string, model string) (string, error) {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return "", nil
	}
	switch agent {
	case "claude":
		switch model {
		case "sonnet", "opus", "haiku":
			return model, nil
		default:
			return "", errors.New("--model for claude must be sonnet, opus, or haiku")
		}
	case "gemini":
		switch model {
		case "gemini", "gemini-3.5", "3.5":
			return "gemini-3.5", nil
		default:
			return "", errors.New("--model for gemini is fixed to gemini-3.5")
		}
	default:
		return "", errors.New("--model is supported only for claude or gemini")
	}
}
