package nodes_test

import (
	"testing"

	"christiangeorgelucas/makefile-tools/axiom"
)

// ── axiom.Context test double (shared by every *_test.go in this package) ────

type testContext struct {
	t            *testing.T
	secretsMap   map[string]string
	revokedNames map[string]bool
}

func newTestContext(t *testing.T) *testContext {
	return &testContext{t: t, secretsMap: map[string]string{}, revokedNames: map[string]bool{}}
}

type testLogger struct{ t *testing.T }

func (l *testLogger) Debug(msg string, args ...any) { l.t.Logf("DEBUG  %s %v", msg, args) }
func (l *testLogger) Info(msg string, args ...any)  { l.t.Logf("INFO   %s %v", msg, args) }
func (l *testLogger) Warn(msg string, args ...any)  { l.t.Logf("WARN   %s %v", msg, args) }
func (l *testLogger) Error(msg string, args ...any) { l.t.Logf("ERROR  %s %v", msg, args) }

type testSecrets struct {
	m       map[string]string
	revoked map[string]bool
}

func (s testSecrets) Get(name string) (string, bool) { v, ok := s.m[name]; return v, ok }

func (s testSecrets) Status(name string) axiom.SecretStatus {
	if _, ok := s.m[name]; ok {
		return axiom.SecretStatusAvailable
	}
	if s.revoked[name] {
		return axiom.SecretStatusRevoked
	}
	return axiom.SecretStatusUnset
}

type testFlowReflection struct{}

func (testFlowReflection) Nodes() []axiom.ReflectionNode     { return nil }
func (testFlowReflection) Edges() []axiom.ReflectionEdge     { return nil }
func (testFlowReflection) LoopEdges() []axiom.ReflectionEdge { return nil }
func (testFlowReflection) Position() axiom.FlowPosition      { return axiom.FlowPosition{} }
func (testFlowReflection) GraphID() string                   { return "" }

type testReflection struct{}

func (testReflection) Flow() axiom.FlowReflection { return testFlowReflection{} }

type testFlowMutation struct{}

func (testFlowMutation) AddNode(_, _ string, _ *axiom.CanvasPosition) uint32 { return 0 }
func (testFlowMutation) AddEdge(_, _ uint32, _ *axiom.EdgeCondition)         {}

type testMutation struct{}

func (testMutation) Flow() axiom.FlowMutation { return testFlowMutation{} }

func (c *testContext) Log() axiom.Logger            { return &testLogger{c.t} }
func (c *testContext) Secrets() axiom.Secrets       { return testSecrets{c.secretsMap, c.revokedNames} }
func (c *testContext) ExecutionID() string          { return "test-execution-id" }
func (c *testContext) FlowID() string               { return "test-flow-id" }
func (c *testContext) TenantID() string             { return "test-tenant-id" }
func (c *testContext) Reflection() axiom.Reflection { return testReflection{} }
func (c *testContext) Mutation() axiom.Mutation     { return testMutation{} }

var _ axiom.Context = (*testContext)(nil)

// ── shared fixtures ────────────────────────────────────────────────────────

// kitchenSinkMakefile exercises every construct this package parses: plain
// and pattern rules, multiple targets on one rule, an inline ";" recipe, a
// double-colon rule, every assignment operator, .PHONY, .DEFAULT_GOAL as a
// variable, "## " and preceding-comment help text, a backslash continuation,
// a trailing "#" comment, ifeq/else/endif, a define/endef block, and
// include/-include directives. Expected values below are hand-derived from
// the GNU Make manual's documented grammar (an oracle independent of this
// package's own code), not by running this package and copying its output.
const kitchenSinkMakefile = `# top-level comment, not attached to anything
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

lib.a::
	@echo building lib variant A
lib.a::
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
