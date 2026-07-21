package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetTargetHelp extracts the documentation associated with one named
// target: a trailing "## help text" comment on its own rule line, and/or a
// comment block directly above it (no blank line in between) — the two
// conventions self-documenting Makefiles use for `make help`. found is
// false when no rule defines that target name.
func GetTargetHelp(ctx context.Context, ax axiom.Context, input *gen.TargetQuery) (*gen.TargetHelp, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	t, found := findTarget(parsed.Targets, input.GetTargetName())
	out := &gen.TargetHelp{TargetName: input.GetTargetName(), Found: found}
	if found {
		out.HelpText = t.HelpComments
	}
	return out, nil
}
