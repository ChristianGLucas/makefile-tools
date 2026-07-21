package nodes

import (
	"context"
	"strings"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListPatternRules lists every static pattern rule — a target whose name
// contains "%", e.g. "%.o: %.c" — in source order, with its prerequisite
// pattern(s) and recipe.
func ListPatternRules(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.PatternRuleList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	var pattern []mkfile.Target
	for _, t := range parsed.Targets {
		if strings.Contains(t.Name, "%") {
			pattern = append(pattern, t)
		}
	}

	return &gen.PatternRuleList{PatternRules: toGenTargets(pattern)}, nil
}
