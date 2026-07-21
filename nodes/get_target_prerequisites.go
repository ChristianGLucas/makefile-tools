package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetTargetPrerequisites extracts one named target's prerequisite (dependency)
// list. found is false when no rule defines that target name.
func GetTargetPrerequisites(ctx context.Context, ax axiom.Context, input *gen.TargetQuery) (*gen.TargetPrerequisites, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	t, found := findTarget(parsed.Targets, input.GetTargetName())
	out := &gen.TargetPrerequisites{TargetName: input.GetTargetName(), Found: found}
	if found {
		out.Prerequisites = t.Prerequisites
	}
	return out, nil
}
