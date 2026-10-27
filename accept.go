package tig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// DefaultAcceptTimeout is how long one acceptance command may run.
const DefaultAcceptTimeout = 10 * time.Minute

// maxOutput is how much of a command's output a step keeps.
const maxOutput = 1 << 20

// RunAcceptance runs an acceptance section's commands in the project folder
// and records each one as a check step that tig ran (these are the only
// checks that count when the project verifies strictly). Only the commands
// written in the section run; nothing else does. With no work in progress,
// the run is recorded as a work of its own.
func (r *Repo) RunAcceptance(ctx context.Context, block, actor string, timeout time.Duration) ([]Step, error) {
	c, err := r.Chain(ctx, "")
	if err != nil {
		return nil, err
	}
	b, err := c.Block(block)
	if err != nil {
		return nil, err
	}
	if !b.IsAcceptance() {
		return nil, fmt.Errorf("tig: %q is not an acceptance section", b.Title)
	}
	if len(b.Accept) == 0 {
		return nil, fmt.Errorf("tig: the acceptance section in %s has no commands", b.Path)
	}
	if timeout <= 0 {
		timeout = DefaultAcceptTimeout
	}
	cur, err := r.Current()
	if err != nil {
		return nil, err
	}
	if cur == nil {
		if _, err := r.StartWork(ctx, []string{b.Hash}, "acceptance of "+acceptsTitle(c, b), actor); err != nil {
			return nil, err
		}
	}
	var steps []Step
	for _, command := range b.Accept {
		out, code := runCommand(ctx, r.Root, command, timeout)
		s, err := r.RecordStep(ctx, StepInput{Kind: StepCheck, Basis: []string{b.Hash}, Command: command, Exit: &code,
			Output: out, Runner: RunnerTig, Actor: actor})
		if err != nil {
			return steps, err
		}
		steps = append(steps, s)
	}
	if cur == nil {
		if _, err := r.FinishWork(ctx, "", nil); err != nil {
			return steps, err
		}
	}
	return steps, nil
}

func acceptsTitle(c *Chain, b *Block) string {
	if p := c.Blocks[b.Parent]; p != nil && p.Level > 0 {
		return p.Title
	}
	return b.Path
}

// runCommand runs one command line in the system shell and returns its
// output (stdout and stderr together, cut at maxOutput) and exit code.
func runCommand(ctx context.Context, dir, command string, timeout time.Duration) (string, int) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := shellCommand(ctx, command)
	cmd.Dir = dir
	var out limitedBuffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	text := out.String()
	if out.cut {
		text += "\n[tig: output cut at 1 MiB]"
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return text, 0
	case ctx.Err() == context.DeadlineExceeded:
		return text + fmt.Sprintf("\n[tig: stopped after %s]", timeout), -1
	case errors.As(err, &exit):
		return text, exit.ExitCode()
	default:
		return text + "\n[tig: " + err.Error() + "]", -1
	}
}

type limitedBuffer struct {
	bytes.Buffer
	cut bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := maxOutput - b.Len(); room < len(p) {
		if room > 0 {
			b.Buffer.Write(p[:room])
		}
		b.cut = true
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
