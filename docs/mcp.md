# MCP server

`tigwx mcp` serves a project's tig to agents over MCP, on stdin and stdout. Code: [mcpserver](../mcpserver/).

## Tools

| Tool | What it does |
|---|---|
| `overview` | Start here: the chain's outline, whether the chain is closed, the counts, and the work in progress. Paged. |
| `search` | Blocks, works, steps and files containing every word. Keyword only: no hit does not mean there is nothing. |
| `read_block` | One block: its text, its sections, the detail documents and files it links to, what was built for it, what it was before it changed, and its latest works. |
| `follow` | Walk the chain from a block, down (sections, detail documents, files) or up. |
| `trail` | The works done for a block, everything under it and its detail documents included, or for a file; newest first. With `steps`, each step and the files it changed. |
| `read_file` | A project file as it is now, or as it was in a tree of the footprint. Paged. |
| `check` | The closure report. It only reads. |
| `start_work`, `record_step`, `finish_work` | Leave a footprint: start a work for some blocks, record each step, finish it. |
| `read_commit` | One commit by its short ID: its intent, files, the sentences naming it, the commits it folds. |
| `relation_set` | What a piece of work needs: from sentences, commits or files, the sentences, commits and files one step away. |
| `mark_sentence` | Say which commits follow a sentence. A sentence the document already says stays there; any other is a guess, written to the inferred document as proposed by an AI. Either way the link is only proposed. An agent cannot confirm an intent or a link. |
| `reverse_marks` | Which lines seem to be about which files, by what the lines point at, and what nothing was found for. It follows fixed rules and writes nothing. |
| `describe_sentence` | Say that a line is about files already there, and why. It is recorded as a proposal and binds nothing until someone agrees. |
| `resolve` | What any reference means: an intent, a line, a commit or an event. |
| `sync_lines` | Bring the ledger up to edited lines: new marks, and the intents on them follow. |
| `fold_commits` | Fold commits under a new parent; lines then show the parent, and nothing disappears. |
| `propose_link` | Propose that a block belongs with a file or another block. It is recorded as inferred. |

Resources: `tig://block/{hash}`, `tig://work/{hash}`, `tig://step/{hash}` and `tig://object/{hash}`.

## How an agent works with it

1. `overview`; then `read_block` and `follow` the blocks the task is about, and `trail` them to see how they were built.
2. `start_work` with those blocks. Change the files, then `record_step` an `edit`: tig records exactly what changed. Run whatever checks the work and record each as a `check`. Leave the `<tig …/>` marks in place when editing a document: `sync_lines` replaces them.
3. `finish_work`, then `mark_sentence`: say which sentences the commit followed. `check`: is the chain still closed, and has any line or its code moved without the other?

Agents working at the same time each pass their own `session`.

Following another agent's footprint is how a second agent builds the same thing again: `trail` with `steps` gives every edit and the tree after it, and `read_file` reads a file in that tree.

Every record carries an actor: the MCP client's name, unless the call names one.

## Safety

- tig runs nothing: an agent cannot make it run a command, since it could edit a document first.
- An agent cannot confirm an intent or accept a commit: those tools are not served. Its sentences are recorded as guesses, by `ai`.
- `read_file` reads only the project's own files: nothing outside the project, nothing Git ignores, nothing in `.git` or `.tig`, nothing through a symbolic link.
- A check an agent reports is recorded as what that agent said, and tig passes no verdict on it.

## Acceptance

The tests in [server_test.go](../mcpserver/server_test.go). Among them, two agents: one builds what the documents design; the other, given only the documents and the footprint, builds it again by following the footprint, and knows it is done when the chain is closed again.

- `go test ./mcpserver/`
