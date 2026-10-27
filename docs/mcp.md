# MCP server

`tigprint mcp` serves a project's tig to agents over MCP, on stdin and stdout. Code: [mcpserver](../mcpserver/).

## Tools

| Tool | What it does |
|---|---|
| `overview` | Start here: the chain's outline with each block's status, whether the chain is closed, the counts, and the work in progress. Paged. |
| `search` | Blocks, works, steps and files containing every word. Keyword only: no hit does not mean there is nothing. |
| `read_block` | One block: its text, status and acceptance, its sections, the detail documents and files it links to, what was built for it, what it was before it changed, and its latest works. |
| `follow` | Walk the chain from a block, down (sections, detail documents, files) or up. |
| `trail` | The works done for a block, everything under it and its detail documents included, or for a file; newest first. With `steps`, each step and the files it changed. |
| `read_file` | A project file as it is now, or as it was in a tree of the footprint. Paged. |
| `check` | The closure report. With `run`, tig runs one acceptance section itself. |
| `start_work`, `record_step`, `finish_work` | Leave a footprint: start a work for some blocks, record each step, finish it. |
| `propose_link` | Propose that a block belongs with a file or another block. It is recorded as inferred. |

Resources: `tig://block/{hash}`, `tig://work/{hash}`, `tig://step/{hash}` and `tig://object/{hash}`.

## How an agent works with it

1. `overview`; then `read_block` and `follow` the blocks the task is about, and `trail` them to see how they were built.
2. `start_work` with those blocks. Change the files, then `record_step` an `edit`: tig records exactly what changed. Run the acceptance commands and record each as a `check`, or use `check` with `run` when the server allows it.
3. `finish_work`, then `check`: is the chain still closed?

Following another agent's footprint is how a second agent builds the same thing again: `trail` with `steps` gives every edit and the tree after it, and `read_file` reads a file in that tree.

Every record carries an actor: the MCP client's name, unless the call names one.

## Safety

- tig runs only the commands written in acceptance sections, and over MCP only when the server was started with `-run`. Without that switch an agent cannot make tig run anything, since it could edit a document first.
- `read_file` reads only the project's own files: nothing outside the project, nothing Git ignores, nothing in `.git` or `.tig`, nothing through a symbolic link.
- A check an agent reports is never recorded as run by tig; with `verify strict`, only tig's own runs count.

## Acceptance

The tests in [server_test.go](../mcpserver/server_test.go). Among them, two agents: one builds what the documents design; the other, given only the documents and the footprint, builds it again by following the footprint, and knows it is done when the chain is closed again.

- `go test ./mcpserver/`
