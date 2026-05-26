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
	Async       bool
	JobRunID    string
	StatusJobID string
	ResultJobID string
	CancelJobID string
}

func Parse(args []string, stdin io.Reader, getwd func() (string, error)) (Request, error) {
	parsed, err := parseArgs(args)
	if err != nil {
		return Request{}, err
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

	if stdin != nil {
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
		if parsed.statusJobID != "" || parsed.resultJobID != "" || parsed.cancelJobID != "" {
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
		Async:       parsed.async,
		JobRunID:    parsed.jobRunID,
		StatusJobID: parsed.statusJobID,
		ResultJobID: parsed.resultJobID,
		CancelJobID: parsed.cancelJobID,
	}, nil
}

type parsedArgs struct {
	cwd         string
	promptFile  string
	json        bool
	agent       string
	fullAccess  bool
	worktree    bool
	profile     string
	effort      string
	model       string
	async       bool
	jobRunID    string
	statusJobID string
	resultJobID string
	cancelJobID string
	positionals []string
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{json: true, agent: "codex", effort: "medium"}
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
			switch args[i] {
			case "medium", "high":
				parsed.effort = args[i]
			default:
				return parsedArgs{}, errors.New("--effort must be medium or high")
			}
		case "--model":
			i++
			if i >= len(args) {
				return parsedArgs{}, errors.New("--model requires a value")
			}
			parsed.model = args[i]
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
		default:
			parsed.positionals = append(parsed.positionals, arg)
		}
	}
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
