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
