# makefile-tools

Deterministic parsing and structural inspection of GNU/BSD Makefiles, built for
the Axiom marketplace (`christiangeorgelucas/makefile-tools`).

Core rule and variable tokenizing is delegated to
[`github.com/checkmake/checkmake/parser`](https://github.com/checkmake/checkmake)
(MIT), the hand-rolled tokenizer behind the `checkmake` Makefile linter. This
package adds backslash-continuation joining, comment/directive extraction,
exact assignment-operator capture, double-colon and pattern-rule detection,
`.DEFAULT_GOAL` resolution, cycle-safe variable expansion, dependency-graph /
build-order derivation, and structural validation on top — plus two verified
fixes to checkmake's own output (an off-by-one in reported variable line
numbers, and a spurious leading `":"` dependency on double-colon rules; see
`internal/mkfile/mkfile.go` and its tests for both).

The Makefile is always supplied as text by the caller: no shell execution, no
`make` invocation, no filesystem access beyond a single self-created,
immediately-removed temp file used to hand sanitized text to the tokenizer, no
network, no wall-clock, no randomness — pure static parsing, bounded to 2 MiB /
50,000 lines, with recursive/cyclic variable expansion detected and reported
rather than looped.

## Use it from your agent or app

Every node in this package is a **live, auto-scaling API endpoint** on the
[Axiom](https://axiomide.com) marketplace — call it from an AI agent or your own
code, with nothing to self-host.

**📦 See it on the marketplace:**
https://dev.axiomide.com/marketplace/christiangeorgelucas/makefile-tools@0.1.0

**Hook it up to an AI agent (MCP).** Add Axiom's hosted MCP server to any MCP
client and every node becomes a typed tool your agent can call — search the
catalog, inspect a schema, and invoke it directly.

```bash
# Claude Code
claude mcp add --transport http axiom https://api.axiomide.com/mcp \
  --header "Authorization: Bearer $AXIOM_API_KEY"
```

Claude Desktop, Cursor, or any config-based client:

```json
{
  "mcpServers": {
    "axiom": {
      "type": "http",
      "url": "https://api.axiomide.com/mcp",
      "headers": { "Authorization": "Bearer YOUR_AXIOM_API_KEY" }
    }
  }
}
```

**Call it from the CLI.**

```bash
axiom invoke christiangeorgelucas/makefile-tools/ParseMakefile --input '{ ... }'
```

**Call it over HTTP.**

```bash
curl -X POST https://api.axiomide.com/invocations/v1/nodes/christiangeorgelucas/makefile-tools/0.1.0/ParseMakefile \
  -H "Authorization: Bearer $AXIOM_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{ ... }'
```

> Input/output schema for each node is on the marketplace page above, or via
> `axiom inspect node christiangeorgelucas/makefile-tools/ParseMakefile`.

### Get started free

Install the CLI:

```bash
# macOS / Linux — Homebrew
brew install axiomide/tap/axiom

# macOS / Linux — install script
curl -fsSL https://raw.githubusercontent.com/AxiomIDE/axiom-releases/main/install.sh | sh
```

**Windows:** download the `windows/amd64` `.zip` from the
[releases page](https://github.com/AxiomIDE/axiom-releases/releases), unzip it,
and put `axiom.exe` on your `PATH`.

Then `axiom version` to verify, `axiom login` (GitHub or Google) to authenticate,
and create an API key under **Console → API Keys**. Docs and sign-up at
**[axiomide.com](https://axiomide.com)**.

## Nodes

- `ParseMakefile` — full structured parse (targets, variables, comments,
  directives, PHONY set, default goal, validation issues).
- `ListTargets` / `GetTargetPrerequisites` / `GetTargetRecipe`
- `ListVariables` / `ExpandVariable` (static `$(VAR)`/`${VAR}` expansion with
  cycle detection)
- `ListPhonyTargets` / `GetDefaultGoal`
- `ListPatternRules` / `ListDoubleColonRules`
- `GetDependencyGraph` (target dependency graph + topological build order +
  cycle detection)
- `ListIncludes` (reported only — never fetched or executed)
- `GetTargetHelp` / `ListTargetHelp` (the `## help text` self-documenting
  Makefile convention)
- `GetSummary`
- `ValidateMakefile` (structural issues with line numbers)

## License

MIT — Copyright (c) 2026 Christian George Lucas.
