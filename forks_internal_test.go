package tig

import "testing"

func TestOpenForks(t *testing.T) {
	ev := func(id, parent, kind, text string) IntentEvent {
		return IntentEvent{ID: id, Parent: parent, Kind: kind, Text: text}
	}
	root := ev("r", "", IntentProposed, "first")
	for _, c := range []struct {
		name string
		log  []IntentEvent
		open int
	}{
		{"one line of events", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("b", "a", IntentRejected, "")}, 0},
		{"the same decision twice", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("b", "r", IntentConfirmed, "")}, 0},
		{"the same words twice", []IntentEvent{root, ev("a", "r", IntentTouched, "new"), ev("b", "r", IntentTouched, "new")}, 0},
		{"two decisions", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("b", "r", IntentRejected, "")}, 1},
		{"two sets of words", []IntentEvent{root, ev("a", "r", IntentTouched, "one"), ev("b", "r", IntentTouched, "two")}, 1},
		{"a text move beside a decision", []IntentEvent{root, ev("a", "r", IntentSpans, ""), ev("b", "r", IntentConfirmed, "")}, 0},
		{"settled by a decision", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("b", "r", IntentRejected, ""), ev("c", "b", IntentConfirmed, "")}, 0},
		{"an edit does not settle a decision", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("b", "r", IntentRejected, ""), ev("c", "b", IntentTouched, "edited")}, 1},
		{"an edit settles a difference in words", []IntentEvent{root, ev("a", "r", IntentTouched, "one"), ev("b", "r", IntentTouched, "two"), ev("c", "b", IntentTouched, "both")}, 0},
		{"a decision on one branch before the other arrived", []IntentEvent{root, ev("a", "r", IntentConfirmed, ""), ev("c", "a", IntentDeprecated, ""), ev("b", "r", IntentRejected, "")}, 1},
	} {
		children := map[string][]IntentEvent{}
		for _, e := range c.log[1:] {
			children[e.Parent] = append(children[e.Parent], e)
		}
		if got := openForks(c.log, children); len(got) != c.open {
			t.Errorf("%s: %d open, want %d: %+v", c.name, len(got), c.open, got)
		}
	}
}
