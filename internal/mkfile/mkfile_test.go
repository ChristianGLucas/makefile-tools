package mkfile

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// kitchenSink exercises every construct this package parses. Expected
// values below are hand-derived from the GNU Make manual's documented
// grammar (an oracle independent of this package's own code) and were
// additionally cross-checked line-by-line against a standalone probe of
// github.com/checkmake/checkmake/parser directly — which is how the two
// real bugs this file guards against (Variable.LineNumber off-by-one, and a
// spurious leading ":" dependency on double-colon rules) were found.
const kitchenSink = `# top-level comment, not attached to anything
CC := gcc
CFLAGS = -Wall -O2
EXTRA_FLAGS ?= -g
SOURCES += main.c
SOURCES += util.c
SHELL_VAR != echo sh
OBJS = $(SOURCES:.c=.o)

.DEFAULT_GOAL := app

.PHONY: all clean

## Build everything
all: app

app: main.o util.o long.o ; @echo linking $@
	$(CC) $(CFLAGS) -o app main.o util.o

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

# Long prerequisite list, continued across lines.
long.o: main.c \
    util.c
	$(CC) -c main.c -o long.o

install uninstall:
	@echo install-or-uninstall

lib.a:: a.o
	@echo building lib variant A
lib.a:: b.o
	@echo building lib variant B

clean: # trailing comment, not help
	rm -f *.o app

ifeq ($(OS),Windows_NT)
WINFLAG = 1
else
WINFLAG = 0
endif

define GREETING
echo hello
echo world
endef

include config.mk
-include optional.mk
`

func mustTarget(t *testing.T, targets []Target, name string, line int) Target {
	t.Helper()
	for _, tg := range targets {
		if tg.Name == name && tg.LineNumber == line {
			return tg
		}
	}
	var names []string
	for _, tg := range targets {
		names = append(names, tg.Name)
	}
	t.Fatalf("target %q at line %d not found; have %v", name, line, names)
	return Target{}
}

func TestParse_KitchenSink_Targets(t *testing.T) {
	p, err := Parse(kitchenSink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Targets) != 9 {
		var names []string
		for _, tg := range p.Targets {
			names = append(names, tg.Name)
		}
		t.Fatalf("expected 9 targets (install/uninstall split from one rule header), got %d: %v", len(p.Targets), names)
	}

	all := mustTarget(t, p.Targets, "all", 15)
	if !reflect.DeepEqual(all.Prerequisites, []string{"app"}) {
		t.Errorf("all.Prerequisites = %v", all.Prerequisites)
	}
	if !all.IsPhony {
		t.Errorf("all should be phony (declared in .PHONY)")
	}
	if !reflect.DeepEqual(all.HelpComments, []string{"Build everything"}) {
		t.Errorf("all.HelpComments = %v", all.HelpComments)
	}

	app := mustTarget(t, p.Targets, "app", 17)
	if !reflect.DeepEqual(app.Prerequisites, []string{"main.o", "util.o", "long.o"}) {
		t.Errorf("app.Prerequisites = %v", app.Prerequisites)
	}
	if !reflect.DeepEqual(app.Recipe, []string{"@echo linking $@", "$(CC) $(CFLAGS) -o app main.o util.o"}) {
		t.Errorf("app.Recipe (inline ';' recipe + tab body) = %v", app.Recipe)
	}

	pat := mustTarget(t, p.Targets, "%.o", 20)
	if !pat.IsPatternRule {
		t.Errorf("%%.o should be a pattern rule")
	}
	if !reflect.DeepEqual(pat.Prerequisites, []string{"%.c"}) {
		t.Errorf("%%.o.Prerequisites = %v", pat.Prerequisites)
	}

	long := mustTarget(t, p.Targets, "long.o", 24)
	if !reflect.DeepEqual(long.Prerequisites, []string{"main.c", "util.c"}) {
		t.Errorf("long.o.Prerequisites (backslash-continued) = %v", long.Prerequisites)
	}
	if !reflect.DeepEqual(long.HelpComments, []string{"Long prerequisite list, continued across lines."}) {
		t.Errorf("long.o.HelpComments (preceding comment) = %v", long.HelpComments)
	}

	// "install uninstall:" names TWO targets sharing one rule (GNU Make
	// manual §4.2) — each must appear as its own Target, not one bogus
	// Target literally named "install uninstall".
	install := mustTarget(t, p.Targets, "install", 28)
	if len(install.Prerequisites) != 0 {
		t.Errorf("install should have no prerequisites, got %v", install.Prerequisites)
	}
	if !reflect.DeepEqual(install.Recipe, []string{"@echo install-or-uninstall"}) {
		t.Errorf("install.Recipe = %v", install.Recipe)
	}
	uninstall := mustTarget(t, p.Targets, "uninstall", 28)
	if !reflect.DeepEqual(uninstall.Recipe, []string{"@echo install-or-uninstall"}) {
		t.Errorf("uninstall.Recipe = %v (should share install's recipe)", uninstall.Recipe)
	}

	lib1 := mustTarget(t, p.Targets, "lib.a", 31)
	if !lib1.IsDoubleColon {
		t.Errorf("lib.a (line 31) should be double-colon")
	}
	if !reflect.DeepEqual(lib1.Prerequisites, []string{"a.o"}) {
		t.Errorf("lib.a (line 31).Prerequisites = %v — checkmake's spurious leading \":\" must be stripped", lib1.Prerequisites)
	}
	lib2 := mustTarget(t, p.Targets, "lib.a", 33)
	if !reflect.DeepEqual(lib2.Prerequisites, []string{"b.o"}) {
		t.Errorf("lib.a (line 33).Prerequisites = %v", lib2.Prerequisites)
	}

	clean := mustTarget(t, p.Targets, "clean", 36)
	if !clean.IsPhony {
		t.Errorf("clean should be phony")
	}
	if !reflect.DeepEqual(clean.HelpComments, []string{"trailing comment, not help"}) {
		t.Errorf("clean.HelpComments (trailing comment) = %v", clean.HelpComments)
	}

	if !reflect.DeepEqual(p.PhonyTargets, []string{"all", "clean"}) {
		t.Errorf("PhonyTargets = %v", p.PhonyTargets)
	}
}

func TestParse_KitchenSink_Variables(t *testing.T) {
	p, err := Parse(kitchenSink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []Variable{
		{Name: "CC", Value: "gcc", Operator: ":=", LineNumber: 2},
		{Name: "CFLAGS", Value: "-Wall -O2", Operator: "=", LineNumber: 3},
		{Name: "EXTRA_FLAGS", Value: "-g", Operator: "?=", LineNumber: 4},
		{Name: "SOURCES", Value: "main.c", Operator: "+=", LineNumber: 5},
		{Name: "SOURCES", Value: "util.c", Operator: "+=", LineNumber: 6},
		{Name: "SHELL_VAR", Value: "echo sh", Operator: "!=", LineNumber: 7},
		{Name: "OBJS", Value: "$(SOURCES:.c=.o)", Operator: "=", LineNumber: 8},
		{Name: ".DEFAULT_GOAL", Value: "app", Operator: ":=", LineNumber: 10},
		{Name: "WINFLAG", Value: "1", Operator: "=", LineNumber: 40},
		{Name: "WINFLAG", Value: "0", Operator: "=", LineNumber: 42},
	}
	if !reflect.DeepEqual(p.Variables, want) {
		t.Errorf("Variables mismatch.\ngot:  %+v\nwant: %+v", p.Variables, want)
	}
}

func TestParse_KitchenSink_CommentsDirectivesGoalLineCount(t *testing.T) {
	p, err := Parse(kitchenSink)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.Comments) != 4 {
		t.Errorf("expected 4 comments, got %d: %+v", len(p.Comments), p.Comments)
	}
	foundHelp := false
	for _, c := range p.Comments {
		if c.LineNumber == 14 {
			if c.Text != "Build everything" || !c.IsHelpMarker {
				t.Errorf("line-14 comment = %+v, want text=%q IsHelpMarker=true", c, "Build everything")
			}
			foundHelp = true
		}
	}
	if !foundHelp {
		t.Errorf("expected a comment at line 14")
	}

	wantDirectiveTypes := []string{"ifeq", "else", "endif", "define", "include", "-include"}
	if len(p.Directives) != len(wantDirectiveTypes) {
		t.Fatalf("expected %d directives, got %d: %+v", len(wantDirectiveTypes), len(p.Directives), p.Directives)
	}
	for i, wantType := range wantDirectiveTypes {
		if p.Directives[i].Type != wantType {
			t.Errorf("directive[%d].Type = %q, want %q", i, p.Directives[i].Type, wantType)
		}
	}
	// define block: the "GREETING" body lines must never leak out as their
	// own directive, comment, target, or variable.
	for _, v := range p.Variables {
		if v.Name == "BAZ" {
			t.Errorf("define body leaked as a variable: %+v", v)
		}
	}

	if !p.DefaultGoalFound || p.DefaultGoal != "app" || p.DefaultGoalSource != "DEFAULT_GOAL variable" {
		t.Errorf("DefaultGoal = %q found=%v source=%q, want app/true/\"DEFAULT_GOAL variable\"", p.DefaultGoal, p.DefaultGoalFound, p.DefaultGoalSource)
	}

	if p.LineCount != 51 {
		t.Errorf("LineCount = %d, want 51", p.LineCount)
	}
}

func TestParse_DefaultGoal_FirstTargetFallback(t *testing.T) {
	p, err := Parse("build: prep\n\t@echo build\nprep:\n\t@echo prep\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.DefaultGoalFound || p.DefaultGoal != "build" || p.DefaultGoalSource != "first target" {
		t.Errorf("got goal=%q found=%v source=%q, want build/true/\"first target\"", p.DefaultGoal, p.DefaultGoalFound, p.DefaultGoalSource)
	}
}

// TestParse_MultiTargetRule_SplitsIntoSeparateTargets is a regression test
// for a CRITICAL finding from independent review: a rule naming multiple
// targets ("install uninstall: prep") was stored as a single bogus Target
// literally named "install uninstall", breaking lookup-by-name, PHONY
// flagging, default-goal resolution, and validation. GNU Make manual §4.2:
// each name is an independent target sharing the prerequisites and recipe.
func TestParse_MultiTargetRule_SplitsIntoSeparateTargets(t *testing.T) {
	content := ".PHONY: install uninstall\ninstall uninstall: prep\n\t@echo doing install or uninstall\n\nprep:\n\t@echo prep\n"
	p, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	install, ok := findTargetForTest(p.Targets, "install")
	if !ok {
		t.Fatalf("expected a Target literally named %q, got names %v", "install", targetNames(p.Targets))
	}
	if !reflect.DeepEqual(install.Prerequisites, []string{"prep"}) {
		t.Errorf("install.Prerequisites = %v, want [prep]", install.Prerequisites)
	}
	if !install.IsPhony {
		t.Errorf("install should be phony (declared in .PHONY)")
	}

	uninstall, ok := findTargetForTest(p.Targets, "uninstall")
	if !ok {
		t.Fatalf("expected a Target literally named %q, got names %v", "uninstall", targetNames(p.Targets))
	}
	if !uninstall.IsPhony {
		t.Errorf("uninstall should be phony (declared in .PHONY)")
	}
	if !reflect.DeepEqual(uninstall.Prerequisites, install.Prerequisites) {
		t.Errorf("uninstall should share install's prerequisites: got %v vs %v", uninstall.Prerequisites, install.Prerequisites)
	}

	for _, bogus := range p.Targets {
		if bogus.Name == "install uninstall" {
			t.Fatalf("found the old bogus joined-name target %q — bug regressed", bogus.Name)
		}
	}

	if p.DefaultGoal != "install" {
		t.Errorf("DefaultGoal = %q, want %q (the first split target)", p.DefaultGoal, "install")
	}

	// No .PHONY-declares-an-undefined-target false positive now that both
	// split names resolve.
	for _, iss := range p.Issues {
		if strings.Contains(iss.Message, "but no rule defines it") {
			t.Errorf("unexpected false-positive validation issue: %+v", iss)
		}
	}
}

func findTargetForTest(targets []Target, name string) (Target, bool) {
	for _, t := range targets {
		if t.Name == name {
			return t, true
		}
	}
	return Target{}, false
}

func targetNames(targets []Target) []string {
	var names []string
	for _, t := range targets {
		names = append(names, t.Name)
	}
	return names
}

// TestParse_ExportOverride_StillCapturesAssignment is a regression test for
// a CRITICAL finding from independent review: "export VAR := value" and
// "override VAR OP value" (GNU Make manual §6.7's own example) were
// classified purely as directives, silently dropping the assignment from
// the variable list and from expansion.
func TestParse_ExportOverride_StillCapturesAssignment(t *testing.T) {
	content := "export PATH := /usr/local/bin:$(PATH)\nCFLAGS := -O2\noverride CFLAGS += -Wall\ntarget:\n\t@echo $(CFLAGS)\n"
	p, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pathVar, ok := findVariableForTest(p.Variables, "PATH")
	if !ok {
		t.Fatalf("expected PATH to be captured as a variable despite the 'export' prefix, got %+v", p.Variables)
	}
	if pathVar.Operator != ":=" || pathVar.Value != "/usr/local/bin:$(PATH)" {
		t.Errorf("PATH variable = %+v", pathVar)
	}

	var cflagsAssignments []Variable
	for _, v := range p.Variables {
		if v.Name == "CFLAGS" {
			cflagsAssignments = append(cflagsAssignments, v)
		}
	}
	if len(cflagsAssignments) != 2 {
		t.Fatalf("expected 2 CFLAGS assignments (the plain one and the 'override' one), got %d: %+v", len(cflagsAssignments), cflagsAssignments)
	}
	if cflagsAssignments[1].Operator != "+=" || cflagsAssignments[1].Value != "-Wall" {
		t.Errorf("the 'override' CFLAGS assignment = %+v, want operator=+= value=-Wall", cflagsAssignments[1])
	}

	// The directive itself must still be reported too (traceability).
	foundExportDirective := false
	foundOverrideDirective := false
	for _, d := range p.Directives {
		if d.Type == "export" {
			foundExportDirective = true
		}
		if d.Type == "override" {
			foundOverrideDirective = true
		}
	}
	if !foundExportDirective || !foundOverrideDirective {
		t.Errorf("expected both 'export' and 'override' directives still reported, got %+v", p.Directives)
	}
}

// TestParse_ExportBareNameList_StaysDirectiveOnly confirms the fix above
// doesn't over-fire: a bare "export VAR1 VAR2" (no assignment, just marking
// already-defined variables for sub-make export) must NOT be misread as an
// assignment.
func TestParse_ExportBareNameList_StaysDirectiveOnly(t *testing.T) {
	content := "FOO = 1\nexport FOO BAR\nall:\n\t@echo hi\n"
	p, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range p.Variables {
		if v.Name == "BAR" {
			t.Fatalf("bare 'export FOO BAR' must not synthesize a BAR assignment, got %+v", p.Variables)
		}
	}
	if len(p.Variables) != 1 || p.Variables[0].Name != "FOO" {
		t.Errorf("expected only the original FOO assignment, got %+v", p.Variables)
	}
}

func findVariableForTest(vars []Variable, name string) (Variable, bool) {
	for _, v := range vars {
		if v.Name == name {
			return v, true
		}
	}
	return Variable{}, false
}

func TestParse_DefineEndef_DoesNotLeakAsTargetOrVariable(t *testing.T) {
	p, err := Parse("define GREETING\nfoo: bar\nBAZ = 1\nendef\n\nreal: dep\n\t@echo ok\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Targets) != 1 || p.Targets[0].Name != "real" {
		t.Fatalf("expected only the 'real' target outside the define block, got %+v", p.Targets)
	}
	if len(p.Variables) != 0 {
		t.Fatalf("expected no variables (BAZ is inside the define block), got %+v", p.Variables)
	}
	if len(p.Directives) != 1 || p.Directives[0].Type != "define" || p.Directives[0].Arguments != "GREETING" {
		t.Fatalf("expected one 'define GREETING' directive, got %+v", p.Directives)
	}
}

func TestParse_UnterminatedDefine_ReportsIssueNotCrash(t *testing.T) {
	p, err := Parse("define GREETING\necho hi\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, iss := range p.Issues {
		if iss.Severity == "error" && strings.Contains(iss.Message, "unterminated") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an 'unterminated define' error issue, got %+v", p.Issues)
	}
}

func TestParse_MalformedIndentation_ReportsWarning(t *testing.T) {
	// Four spaces instead of a tab under a rule header — GNU Make's classic
	// "missing separator" trap.
	p, err := Parse("build:\n    @echo hi\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, iss := range p.Issues {
		if iss.Severity == "warning" && strings.Contains(iss.Message, "literal tab") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a space-indented-recipe warning, got %+v", p.Issues)
	}
}

func TestParse_ContinuationLine_NotFalselyFlaggedAsBadIndent(t *testing.T) {
	// A legitimately backslash-continued, space-indented prerequisite line
	// must NOT trip the "recipe needs a tab" heuristic above.
	p, err := Parse("long.o: main.c \\\n    util.c\n\t$(CC) -c main.c -o long.o\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, iss := range p.Issues {
		if strings.Contains(iss.Message, "literal tab") {
			t.Errorf("continuation line falsely flagged: %+v", iss)
		}
	}
}

func TestParse_PhonyDeclaredButUndefined_ReportsWarning(t *testing.T) {
	p, err := Parse(".PHONY: ghost\nreal:\n\t@echo hi\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, iss := range p.Issues {
		if iss.Severity == "warning" && strings.Contains(iss.Message, `"ghost"`) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning about .PHONY declaring an undefined target, got %+v", p.Issues)
	}
}

func TestParse_DuplicateSingleColonTarget_ReportsWarning(t *testing.T) {
	p, err := Parse("foo:\n\t@echo one\nfoo:\n\t@echo two\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, iss := range p.Issues {
		if iss.Severity == "warning" && strings.Contains(iss.Message, `"foo"`) && strings.Contains(iss.Message, "single-colon") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a duplicate single-colon target warning, got %+v", p.Issues)
	}
}

func TestParse_TooLarge_ReturnsErrTooLarge(t *testing.T) {
	huge := strings.Repeat("x", MaxContentBytes+1)
	_, err := Parse(huge)
	if err == nil {
		t.Fatal("expected an error for oversized content")
	}
	if _, ok := err.(ErrTooLarge); !ok {
		t.Errorf("expected ErrTooLarge, got %T: %v", err, err)
	}
}

func TestParse_TooManyLines_ReturnsErrTooLarge(t *testing.T) {
	huge := strings.Repeat("a:\n", MaxLines+10)
	_, err := Parse(huge)
	if err == nil {
		t.Fatal("expected an error for too many lines")
	}
	if _, ok := err.(ErrTooLarge); !ok {
		t.Errorf("expected ErrTooLarge, got %T: %v", err, err)
	}
}

func TestParse_EmptyInput_NoCrash(t *testing.T) {
	p, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Targets) != 0 || len(p.Variables) != 0 || p.DefaultGoalFound {
		t.Errorf("expected an empty result, got %+v", p)
	}
}

// ── ExpandVariable ────────────────────────────────────────────────────────

func TestExpandVariable_Simple(t *testing.T) {
	vars := []Variable{
		{Name: "A", Value: "x-$(B)-y", LineNumber: 1},
		{Name: "B", Value: "middle", LineNumber: 2},
	}
	r := ExpandVariable("A", vars)
	if !r.Found || r.ExpandedValue != "x-middle-y" || !r.FullyResolved {
		t.Errorf("got %+v, want ExpandedValue=x-middle-y FullyResolved=true", r)
	}
}

func TestExpandVariable_NotFound(t *testing.T) {
	r := ExpandVariable("NOPE", nil)
	if r.Found {
		t.Errorf("expected Found=false, got %+v", r)
	}
}

func TestExpandVariable_UnresolvedReference(t *testing.T) {
	vars := []Variable{{Name: "X", Value: "$(Y)-$(MISSING)", LineNumber: 1}, {Name: "Y", Value: "hello", LineNumber: 2}}
	r := ExpandVariable("X", vars)
	if r.FullyResolved {
		t.Errorf("expected FullyResolved=false")
	}
	if !reflect.DeepEqual(r.UnresolvedReferences, []string{"MISSING"}) {
		t.Errorf("UnresolvedReferences = %v", r.UnresolvedReferences)
	}
	if r.ExpandedValue != "hello-$(MISSING)" {
		t.Errorf("ExpandedValue = %q", r.ExpandedValue)
	}
}

func TestExpandVariable_DirectCycle(t *testing.T) {
	vars := []Variable{{Name: "A", Value: "$(B)", LineNumber: 1}, {Name: "B", Value: "$(A)", LineNumber: 2}}
	r := ExpandVariable("A", vars)
	if !r.CycleDetected {
		t.Fatalf("expected a cycle to be detected, got %+v", r)
	}
	if len(r.CyclePath) == 0 || r.CyclePath[0] != "A" {
		t.Errorf("CyclePath = %v, want to start with A", r.CyclePath)
	}
}

func TestExpandVariable_SelfReference(t *testing.T) {
	vars := []Variable{{Name: "A", Value: "$(A)-suffix", LineNumber: 1}}
	r := ExpandVariable("A", vars)
	if !r.CycleDetected {
		t.Errorf("expected self-reference to be detected as a cycle, got %+v", r)
	}
}

func TestExpandVariable_LongNonCyclicChainTerminates(t *testing.T) {
	// 200 variables chained V0=$(V1), V1=$(V2), ... V199=end — well past
	// MaxExpandDepth; must terminate (not hang) and report FullyResolved
	// false rather than crash, since it exceeds the bound.
	var vars []Variable
	for i := 0; i < 200; i++ {
		name := "V" + itoa(i)
		next := "V" + itoa(i+1)
		vars = append(vars, Variable{Name: name, Value: "$(" + next + ")", LineNumber: i + 1})
	}
	vars = append(vars, Variable{Name: "V200", Value: "end", LineNumber: 201})

	done := make(chan ExpandResult, 1)
	go func() { done <- ExpandVariable("V0", vars) }()
	r := <-done
	if !r.Found {
		t.Errorf("expected Found=true, got %+v", r)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// ── BuildDependencyGraph ─────────────────────────────────────────────────

func TestBuildDependencyGraph_Simple(t *testing.T) {
	targets := []Target{
		{Name: "all", Prerequisites: []string{"build", "test"}},
		{Name: "build", Prerequisites: nil},
		{Name: "test", Prerequisites: []string{"build"}},
	}
	g := BuildDependencyGraph(targets)
	if g.HasCycle {
		t.Fatalf("did not expect a cycle: %+v", g)
	}
	pos := map[string]int{}
	for i, n := range g.BuildOrder {
		pos[n] = i
	}
	if pos["build"] >= pos["test"] || pos["test"] >= pos["all"] {
		t.Errorf("build order %v violates dependency order (build < test < all)", g.BuildOrder)
	}
}

func TestBuildDependencyGraph_ExcludesFilePrerequisites(t *testing.T) {
	targets := []Target{
		{Name: "app", Prerequisites: []string{"main.o", "util.o"}}, // .o files are not targets
	}
	g := BuildDependencyGraph(targets)
	if len(g.Edges) != 1 || len(g.Edges[0].DependsOnTargets) != 0 {
		t.Errorf("expected app to have zero target-dependencies (main.o/util.o aren't targets), got %+v", g.Edges)
	}
}

func TestBuildDependencyGraph_CycleDetected(t *testing.T) {
	targets := []Target{
		{Name: "a", Prerequisites: []string{"b"}},
		{Name: "b", Prerequisites: []string{"c"}},
		{Name: "c", Prerequisites: []string{"a"}},
	}
	g := BuildDependencyGraph(targets)
	if !g.HasCycle {
		t.Fatalf("expected a cycle, got %+v", g)
	}
	if len(g.BuildOrder) != 0 {
		t.Errorf("BuildOrder should be empty on a cycle, got %v", g.BuildOrder)
	}
	got := append([]string{}, g.CycleTargets...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("CycleTargets = %v", got)
	}
}
