# Chain

How tig reads a project's documents as one chain, and when the chain is closed.

## Text blocks

tig splits a Markdown document at its headings (`#` to `######`; headings inside code fences do not count). A block is a heading and its own text, up to the next heading. The text before the first heading is the document's own block. Code: [markdown.go](../markdown.go).

A block is named by a hash of where it sits (the file and the headings above it) and what it says (its own text). Editing a block, or a heading above it, makes a new block; a child's text is not part of its parent's name, so editing a child leaves the parent as it was. Anywhere tig takes a block, a unique prefix of 8 or more characters of its hash works.

## Lineage

When a snapshot changes a document, tig pairs the blocks before and after by fixed rules, in this order, and records the result with the snapshot:

1. **edit**: the same headings, the text changed;
2. **move**: the same text, somewhere else;
3. **split** and **merge**: an old block that two or more new, smaller blocks are mostly made of, or the other way round;
4. **rewrite**: both changed, the text still at least half the same (by the Dice similarity of neighbouring characters, which works the same for text with and without spaces);
5. a single new block mostly made of an old one was split from it; an old block mostly found in a new one was merged into it.

The rest was added or removed. There is no model involved: anyone recomputes the same lineage from the same trees. This is how the footprint of a block follows it through edits. Code: [lineage.go](../lineage.go).

## The chain

The chain starts at the main document (`root` in `.tig/config`, `README.md` unless set) and follows every link to another Markdown file in the project: those are its detail documents, and theirs. A link to any other file, or folder, makes it part of the design of the block whose text links it. A Markdown file the chain never reaches is an orphan. Paths listed as `outside` in the config are not part of the chain. Code: [chain.go](../chain.go).

## Acceptance sections

A section titled `Acceptance` (or `驗收`) holds the commands that accept the part of the document it sits in: each line of a code fence in it, and each list item that starts with `inline code`. tig runs them in the project folder, with `sh -c` or, on Windows, `cmd /c`, and records each one as a check step that tig ran. tig runs nothing but these commands. Code: [accept.go](../accept.go), [shell_other.go](../shell_other.go), [shell_windows.go](../shell_windows.go).

What accepts a block is found by walking up from it. The first that comes wins:

- an acceptance section directly under a block accepts that block and everything under it;
- a block whose own text links to detail documents hands its design on to them, and they need their own acceptance.

A block with neither but with sections under it, an introduction for example, is covered by its sections. A block with none of these is open.

## Closure

`tigprint check`, and the `check` tool over MCP, report:

- **broken**: links to files that do not exist, and orphan documents;
- **open**: design no acceptance covers, detail documents without acceptance or without anything in them, and acceptance sections without commands;
- **stale**: a file built for a block whose design has changed since while the file did not follow; a built file changed outside the footprint, or gone; a proposed link whose file has changed. Code: [links.go](../links.go);
- **no design reaches**: files no block links to and no work built.

The chain is **closed** when there is none of these: every piece of design is covered by an acceptance. A covered block is a **direction** until its acceptance passes, then a **guideline**. A closed chain can be handed to an agent as it is: the directions say what to build, the guidelines what was built and holds.

Which checks count is set by `verify` in `.tig/config`: `loose`, the default, counts the checks agents report too; `strict` counts only the checks tig ran itself.

## Search

Blocks, works, steps and files containing every word of a query. It is keyword search only: no hit does not mean the project has nothing on it. Code: [search.go](../search.go).

## Acceptance

The tests in [markdown_test.go](../markdown_test.go) and [chain_test.go](../chain_test.go):

- `go test -run "Blocks|Lineage|Chain|Stale|Cover|Open" .`
