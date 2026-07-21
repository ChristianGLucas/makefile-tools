package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListTargets lists every target defined in a Makefile, in source order,
// each with its prerequisites, recipe, and PHONY / double-colon /
// pattern-rule flags.
func ListTargets(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.TargetList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	return &gen.TargetList{Targets: toGenTargets(parsed.Targets)}, nil
}
