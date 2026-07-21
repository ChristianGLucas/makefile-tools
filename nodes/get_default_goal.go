package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetDefaultGoal identifies the target `make` would build with no arguments:
// the value of a ".DEFAULT_GOAL" variable assignment if one is present and
// non-empty, otherwise the first ordinary (non-special, non-pattern) target
// defined in the file, per GNU Make's own rule. found is false when neither
// exists.
func GetDefaultGoal(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.DefaultGoal, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	return &gen.DefaultGoal{
		TargetName: parsed.DefaultGoal,
		Found:      parsed.DefaultGoalFound,
		Source:     parsed.DefaultGoalSource,
	}, nil
}
