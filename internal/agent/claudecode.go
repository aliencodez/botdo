package agent

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aliencodez/botdo/internal/model"
)

// bufSize is the scanner buffer size for long log lines (1 MB).
const bufSize = 1 << 20

// ClaudeCodeAdapter spawns the claude CLI to execute tasks.
type ClaudeCodeAdapter struct {
	ClaudeBin string // path to claude binary, default "claude"
}

func (a *ClaudeCodeAdapter) AgentType() string { return "claude-code" }

func (a *ClaudeCodeAdapter) Run(ctx context.Context, taskID int64, prompt, workDir string, perms model.Permissions, out LineWriter) error {
	bin := a.ClaudeBin
	if bin == "" {
		bin = "claude"
	}

	args := []string{"--print", "--permission-mode", "acceptEdits", "--output-format", "stream-json", "--verbose"}

	// Build --allowedTools from permissions. Write and Edit are not permitted
	// by default in non-interactive mode; only add them when explicitly enabled.
	var allowed []string
	if perms.AllowWrite {
		allowed = append(allowed, "Write")
	}
	if perms.AllowEdit {
		allowed = append(allowed, "Edit")
	}
	if len(allowed) > 0 {
		args = append(args, "--allowedTools", strings.Join(allowed, ","))
	}

	// Pass the prompt via stdin so multi-line content and special characters
	// are handled correctly regardless of the shell or CLI version.
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(prompt)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, bufSize), bufSize)
		for scanner.Scan() {
			out.WriteLine("[stderr] " + scanner.Text())
		}
		close(done)
	}()

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, bufSize), bufSize)
	for scanner.Scan() {
		out.WriteLine(scanner.Text())
	}

	<-done

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude exited: %w", err)
	}
	return nil
}
