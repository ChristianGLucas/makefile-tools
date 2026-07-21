package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ValidateMakefile checks basic structural correctness and reports issues
// with 1-based line numbers: recipe lines indented with spaces instead of a
// literal tab (GNU Make's most common "missing separator" gotcha), a
// ".PHONY" entry with no matching rule, more than one single-colon rule for
// the same target (only the last recipe wins), and an unterminated
// "define"/"endef" block. valid is true iff no "error"-severity issue was
// found; "warning"-severity issues do not affect it. This never executes
// anything and cannot detect issues only GNU Make itself would catch (a
// missing included file, an undefined function, a shell syntax error).
func ValidateMakefile(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.ValidationResult, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	valid := true
	for _, issue := range parsed.Issues {
		if issue.Severity == "error" {
			valid = false
			break
		}
	}

	return &gen.ValidationResult{
		Valid:  valid,
		Issues: toGenIssues(parsed.Issues),
	}, nil
}
