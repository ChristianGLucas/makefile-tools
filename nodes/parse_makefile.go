package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ParseMakefile parses a Makefile's full text into its complete structured
// representation: every target with its prerequisites, recipe, and PHONY /
// double-colon / pattern-rule flags; every variable assignment with its
// exact operator; every comment and directive; the PHONY set; the default
// goal; and a line-numbered list of structural issues found along the way.
// This is the general-purpose result every other node in this package is a
// specialized view of.
func ParseMakefile(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.ParsedMakefile, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	return &gen.ParsedMakefile{
		Targets:          toGenTargets(parsed.Targets),
		Variables:        toGenVariables(parsed.Variables),
		Comments:         toGenComments(parsed.Comments),
		Directives:       toGenDirectives(parsed.Directives),
		PhonyTargets:     parsed.PhonyTargets,
		DefaultGoal:      parsed.DefaultGoal,
		DefaultGoalFound: parsed.DefaultGoalFound,
		Issues:           toGenIssues(parsed.Issues),
		LineCount:        int32(parsed.LineCount),
	}, nil
}
