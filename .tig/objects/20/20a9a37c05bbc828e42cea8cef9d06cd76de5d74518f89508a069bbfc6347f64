package tig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// A footprint is the works done on a project. A work is one construction
// (the main record, like a commit): what it was for, the snapshots before and
// after, and its steps in order. A step is one thing done in it (the
// detail): a file edit, a command, an install, a check, a decision.
//
// Every footprint record is written once and never changed. A work lists its
// steps by hash, so the steps are written first and the work last; until
// then the work in progress lives in .tig/WORK, which is local.

// Step kinds.
const (
	StepEdit     = "edit"
	StepCommand  = "command"
	StepInstall  = "install"
	StepCheck    = "check"
	StepDecision = "decision"
	StepNote     = "note"
)

// Work results.
const (
	ResultPassed     = "passed"
	ResultFailed     = "failed"
	ResultDone       = "done" // finished without a check
	ResultRolledBack = "rolled-back"
	ResultAbandoned  = "abandoned"
)

// RunnerTig marks a check tig ran itself (only these count when the project
// verifies strictly).
const RunnerTig = "tig"

// Step is one thing done in a work.
type Step struct {
	ID      string              `json:"id"`
	Kind    string              `json:"kind"`
	Basis   []string            `json:"basis,omitempty"` // the text blocks it was done for
	Before  string              `json:"before,omitempty"`
	After   string              `json:"after,omitempty"` // trees around an edit
	Command string              `json:"command,omitempty"`
	Exit    *int                `json:"exit,omitempty"`
	Output  string              `json:"output,omitempty"` // object: the command's output
	Detail  string              `json:"detail,omitempty"` // object: a decision's question and answer, or a long note
	Note    string              `json:"note,omitempty"`
	Runner  string              `json:"runner,omitempty"` // "tig" when tig ran the check itself
	Actor   string              `json:"actor,omitempty"`
	At      time.Time           `json:"at"`
	Extra   map[string][]string `json:"extra,omitempty"`
}

// Passed reports whether a command or check ended with exit code 0.
func (s Step) Passed() bool { return s.Exit != nil && *s.Exit == 0 }

// Work is one construction.
type Work struct {
	ID        string              `json:"id"`
	Parent    string              `json:"parent,omitempty"` // the work before it
	Basis     []string            `json:"basis,omitempty"`
	Intent    string              `json:"intent"`
	Before    string              `json:"before,omitempty"` // snapshots
	After     string              `json:"after,omitempty"`
	Steps     []string            `json:"steps,omitempty"`
	Result    string              `json:"result"`
	Actor     string              `json:"actor,omitempty"`
	Approvers []string            `json:"approvers,omitempty"`
	Started   time.Time           `json:"started"`
	At        time.Time           `json:"at"`
	Extra     map[string][]string `json:"extra,omitempty"`
}

const (
	stepHeader = "tig-step 1"
	workHeader = "tig-work 1"
)

// PutStep stores a step and returns it with its ID.
func (r *Repo) PutStep(ctx context.Context, s Step) (Step, error) {
	if s.At.IsZero() {
		s.At = time.Now().UTC()
	}
	rec := newRecord(stepHeader)
	rec.set("kind", s.Kind)
	rec.setAll("basis", s.Basis)
	rec.set("before", s.Before)
	rec.set("after", s.After)
	rec.set("command", s.Command)
	if s.Exit != nil {
		rec.set("exit", strconv.Itoa(*s.Exit))
	}
	rec.set("output", s.Output)
	rec.set("detail", s.Detail)
	rec.set("note", s.Note)
	rec.set("runner", s.Runner)
	rec.set("actor", s.Actor)
	rec.setTime("at", s.At)
	rec.setExtra(s.Extra)
	id, err := r.Objects.PutObject(ctx, rec.bytes())
	s.ID = id
	return s, err
}

// LoadStep reads a step.
func LoadStep(ctx context.Context, objects Objects, id string) (Step, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return Step{}, err
	}
	rec, err := parseRecord(raw, stepHeader)
	if err != nil {
		return Step{}, fmt.Errorf("tig: %s is not a step", id)
	}
	s := Step{ID: id, Kind: rec.get("kind"), Basis: rec.all("basis"), Before: rec.get("before"), After: rec.get("after"),
		Command: rec.get("command"), Output: rec.get("output"), Detail: rec.get("detail"), Note: rec.get("note"),
		Runner: rec.get("runner"), Actor: rec.get("actor"), At: rec.time("at"), Extra: rec.extra()}
	if v := rec.get("exit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s.Exit = &n
		}
	}
	if err := hashesValid(id, s.Basis, s.Before, s.After, s.Output, s.Detail); err != nil {
		return Step{}, err
	}
	return s, nil
}

// PutWork stores a work on top of the latest one and makes it the latest.
func (r *Repo) PutWork(ctx context.Context, w Work) (Work, error) {
	if r.Dir != "" {
		unlock, err := lockRepo(ctx, r.Dir)
		if err != nil {
			return Work{}, err
		}
		defer unlock()
	}
	return r.putWork(ctx, w)
}

func (r *Repo) putWork(ctx context.Context, w Work) (Work, error) {
	parent, err := r.readHead(headWorks)
	if err != nil {
		return Work{}, err
	}
	w.Parent = parent
	if w.At.IsZero() {
		w.At = time.Now().UTC()
	}
	rec := newRecord(workHeader)
	rec.set("parent", w.Parent)
	rec.setAll("basis", w.Basis)
	rec.set("intent", w.Intent)
	rec.set("before", w.Before)
	rec.set("after", w.After)
	rec.setAll("step", w.Steps)
	rec.set("result", w.Result)
	rec.set("actor", w.Actor)
	rec.setAll("approver", w.Approvers)
	rec.setTime("started", w.Started)
	rec.setTime("at", w.At)
	rec.setExtra(w.Extra)
	id, err := r.Objects.PutObject(ctx, rec.bytes())
	if err != nil {
		return Work{}, err
	}
	w.ID = id
	return w, r.writeHead(headWorks, id)
}

// LoadWork reads a work.
func LoadWork(ctx context.Context, objects Objects, id string) (Work, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return Work{}, err
	}
	rec, err := parseRecord(raw, workHeader)
	if err != nil {
		return Work{}, fmt.Errorf("tig: %s is not a work", id)
	}
	w := Work{ID: id, Parent: rec.get("parent"), Basis: rec.all("basis"), Intent: rec.get("intent"),
		Before: rec.get("before"), After: rec.get("after"), Steps: rec.all("step"), Result: rec.get("result"),
		Actor: rec.get("actor"), Approvers: rec.all("approver"), Started: rec.time("started"), At: rec.time("at"), Extra: rec.extra()}
	if err := hashesValid(id, append(append([]string{}, w.Basis...), w.Steps...), w.Parent, w.Before, w.After); err != nil {
		return Work{}, err
	}
	return w, nil
}

func hashesValid(id string, list []string, single ...string) error {
	for _, h := range append(list, single...) {
		if h != "" && !ValidHash(h) {
			return fmt.Errorf("tig: record %s has an invalid hash %q", id, h)
		}
	}
	return nil
}

// Works walks the works from the latest, newest first, at most n (n<=0: all).
func (r *Repo) Works(ctx context.Context, n int) ([]Work, error) {
	id, err := r.readHead(headWorks)
	if err != nil {
		return nil, err
	}
	var out []Work
	seen := map[string]bool{}
	for id != "" && !seen[id] && (n <= 0 || len(out) < n) {
		seen[id] = true
		w, err := LoadWork(ctx, r.Objects, id)
		if err != nil {
			return out, err
		}
		out = append(out, w)
		id = w.Parent
	}
	return out, nil
}

// Open works: one work at a time is in progress in a project.

// OpenWork is the work in progress (.tig/WORK).
type OpenWork struct {
	Basis   []string  `json:"basis,omitempty"`
	Intent  string    `json:"intent"`
	Actor   string    `json:"actor,omitempty"`
	Before  string    `json:"before"` // snapshot
	Tree    string    `json:"tree"`   // the tree after the last step
	Steps   []string  `json:"steps,omitempty"`
	Started time.Time `json:"started"`
}

// ErrNoWork is returned when a step is recorded with no work in progress.
var ErrNoWork = errors.New("tig: no work in progress (start one first)")

// ErrWorkOpen is returned when a work is started while another one is in
// progress.
var ErrWorkOpen = errors.New("tig: a work is already in progress (finish it first)")

func (r *Repo) workFile() string { return filepath.Join(r.Dir, "WORK") }

// Current is the work in progress, if any.
func (r *Repo) Current() (*OpenWork, error) {
	if r.Dir == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(r.workFile())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var w OpenWork
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, fmt.Errorf("tig: .tig/WORK: %w", err)
	}
	return &w, nil
}

func (r *Repo) saveCurrent(w *OpenWork) error {
	raw, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(r.workFile(), raw, 0o644)
}

// StartWork begins a work: it takes a snapshot of the project as it is.
func (r *Repo) StartWork(ctx context.Context, basis []string, intent, actor string) (*OpenWork, error) {
	if r.Dir == "" {
		return nil, errors.New("tig: works need a .tig folder")
	}
	if err := hashesValid("work", basis); err != nil {
		return nil, err
	}
	unlock, err := lockWork(ctx, r.Dir)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if cur, err := r.Current(); err != nil || cur != nil {
		if err == nil {
			err = ErrWorkOpen
		}
		return nil, err
	}
	s, _, err := r.Take(ctx, ReasonBefore, actor)
	if err != nil {
		return nil, err
	}
	w := &OpenWork{Basis: basis, Intent: intent, Actor: actor, Before: s.ID, Tree: s.Tree, Started: time.Now().UTC()}
	return w, r.saveCurrent(w)
}

// StepInput is one step of the work in progress. For an edit, tig takes the
// trees itself: the project after the last step, and the project now.
type StepInput struct {
	Kind    string
	Basis   []string
	Command string
	Exit    *int
	Output  string // the command's output, stored as an object
	Detail  string // a decision's question and answer, or a long note, stored as an object
	Note    string
	Runner  string
	Actor   string
}

// RecordStep adds a step to the work in progress.
func (r *Repo) RecordStep(ctx context.Context, in StepInput) (Step, error) {
	if r.Dir == "" {
		return Step{}, ErrNoWork
	}
	unlock, err := lockWork(ctx, r.Dir)
	if err != nil {
		return Step{}, err
	}
	defer unlock()
	cur, err := r.Current()
	if err != nil {
		return Step{}, err
	}
	if cur == nil {
		return Step{}, ErrNoWork
	}
	if err := hashesValid("step", in.Basis); err != nil {
		return Step{}, err
	}
	s := Step{Kind: in.Kind, Basis: in.Basis, Command: in.Command, Exit: in.Exit, Note: in.Note, Runner: in.Runner, Actor: in.Actor}
	if s.Actor == "" {
		s.Actor = cur.Actor
	}
	if len(s.Basis) == 0 {
		s.Basis = cur.Basis
	}
	if in.Output != "" {
		if s.Output, err = r.Objects.PutObject(ctx, []byte(in.Output)); err != nil {
			return Step{}, err
		}
	}
	if in.Detail != "" {
		if s.Detail, err = r.Objects.PutObject(ctx, []byte(in.Detail)); err != nil {
			return Step{}, err
		}
	}
	// Every step that can change files records the trees around it, so the
	// footprint shows exactly what each step did.
	if s.Kind == StepEdit || s.Kind == StepCommand || s.Kind == StepInstall {
		files, err := r.Scan(ctx)
		if err != nil {
			return Step{}, err
		}
		tree, err := WriteTree(ctx, r.Objects, files)
		if err != nil {
			return Step{}, err
		}
		if tree != cur.Tree || s.Kind == StepEdit {
			s.Before, s.After = cur.Tree, tree
			cur.Tree = tree
		}
	}
	if s, err = r.PutStep(ctx, s); err != nil {
		return Step{}, err
	}
	cur.Steps = append(cur.Steps, s.ID)
	return s, r.saveCurrent(cur)
}

// FinishWork ends the work in progress: it takes a snapshot of the project
// and writes the work. With no result given, the result comes from its
// checks: passed when each command's last check passed, failed when one did
// not, and "done" when there was no check.
func (r *Repo) FinishWork(ctx context.Context, result string, approvers []string) (Work, error) {
	if r.Dir == "" {
		return Work{}, ErrNoWork
	}
	unlock, err := lockWork(ctx, r.Dir)
	if err != nil {
		return Work{}, err
	}
	defer unlock()
	cur, err := r.Current()
	if err != nil {
		return Work{}, err
	}
	if cur == nil {
		return Work{}, ErrNoWork
	}
	after, _, err := r.Take(ctx, ReasonAfter, cur.Actor)
	if err != nil {
		return Work{}, err
	}
	if result == "" {
		last := map[string]bool{}
		for _, id := range cur.Steps {
			if s, err := LoadStep(ctx, r.Objects, id); err == nil && s.Kind == StepCheck {
				last[s.Command] = s.Passed()
			}
		}
		result = ResultDone
		if len(last) > 0 {
			result = ResultPassed
		}
		for _, ok := range last {
			if !ok {
				result = ResultFailed
			}
		}
	}
	w := Work{Basis: cur.Basis, Intent: cur.Intent, Before: cur.Before, After: after.ID, Steps: cur.Steps,
		Result: result, Actor: cur.Actor, Approvers: approvers, Started: cur.Started}
	if w, err = r.PutWork(ctx, w); err != nil {
		return Work{}, err
	}
	return w, os.Remove(r.workFile())
}
