package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetSummary counts a Makefile's targets, variables, PHONY declarations,
// pattern rules, double-colon rules, comments, directives, and source
// lines — a quick shape check before deciding which detailed node to call.
func GetSummary(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.MakefileSummary, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	var patternCount, doubleCount int
	for _, t := range parsed.Targets {
		if t.IsPatternRule {
			patternCount++
		}
		if t.IsDoubleColon {
			doubleCount++
		}
	}

	return &gen.MakefileSummary{
		TargetCount:      int32(len(parsed.Targets)),
		VariableCount:    int32(len(parsed.Variables)),
		PhonyCount:       int32(len(parsed.PhonyTargets)),
		PatternRuleCount: int32(patternCount),
		DoubleColonCount: int32(doubleCount),
		CommentCount:     int32(len(parsed.Comments)),
		DirectiveCount:   int32(len(parsed.Directives)),
		LineCount:        int32(parsed.LineCount),
	}, nil
}
