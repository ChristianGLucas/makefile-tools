package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListDoubleColonRules lists every rule declared with "::" instead of ":".
// Double-colon rules let multiple independent rules build the same target,
// each running when its own prerequisites are newer — a distinct GNU Make
// feature from an ordinary single-colon rule with multiple prerequisites.
func ListDoubleColonRules(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.DoubleColonRuleList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	var doubles []mkfile.Target
	for _, t := range parsed.Targets {
		if t.IsDoubleColon {
			doubles = append(doubles, t)
		}
	}

	return &gen.DoubleColonRuleList{DoubleColonRules: toGenTargets(doubles)}, nil
}
