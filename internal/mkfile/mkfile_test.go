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
	if len(p.Targets) != 8 {
		var names []string
		for _, tg := range p.Targets {
			names = append(names, tg.Name)
		}
		t.Fatalf("expected 8 targets, got %d: %v", len(p.Targets), names)
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

	multi := mustTarget(t, p.Targets, "install uninstall", 28)
	if len(multi.Prerequisites) != 0 {
		t.Errorf("install uninstall should have no prerequisites, got %v", multi.Prerequisites)
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
