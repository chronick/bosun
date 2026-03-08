package claim

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/chronick/bosun/internal/config"
)

func TestParseReadyOutput_Typical(t *testing.T) {
	output := `ID          PRI  TYPE   TITLE
bosun-3hj   1    task   run command - full lifecycle loop
bosun-138   2    task   CLI scaffold with cobra
bosun-216   2    task   claim command - pick highest priority ready task`

	tasks := ParseReadyOutput(output)
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Check first task.
	if tasks[0].ID != "bosun-3hj" {
		t.Errorf("task 0 ID = %q, want %q", tasks[0].ID, "bosun-3hj")
	}
	if tasks[0].Priority != 1 {
		t.Errorf("task 0 Priority = %d, want 1", tasks[0].Priority)
	}
	if tasks[0].Type != "task" {
		t.Errorf("task 0 Type = %q, want %q", tasks[0].Type, "task")
	}
	if tasks[0].Title != "run command - full lifecycle loop" {
		t.Errorf("task 0 Title = %q, want %q", tasks[0].Title, "run command - full lifecycle loop")
	}

	// Check second task.
	if tasks[1].ID != "bosun-138" {
		t.Errorf("task 1 ID = %q, want %q", tasks[1].ID, "bosun-138")
	}
	if tasks[1].Priority != 2 {
		t.Errorf("task 1 Priority = %d, want 2", tasks[1].Priority)
	}
}

func TestParseReadyOutput_WithPPrefix(t *testing.T) {
	output := `bosun-abc   P0   bug    Critical crash fix
bosun-def   P3   chore  Update dependencies`

	tasks := ParseReadyOutput(output)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Priority != 0 {
		t.Errorf("task 0 Priority = %d, want 0", tasks[0].Priority)
	}
	if tasks[1].Priority != 3 {
		t.Errorf("task 1 Priority = %d, want 3", tasks[1].Priority)
	}
}

func TestParseReadyOutput_Empty(t *testing.T) {
	tasks := ParseReadyOutput("")
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestParseReadyOutput_OnlyHeader(t *testing.T) {
	output := "ID          PRI  TYPE   TITLE\n─────────────────────────────"
	tasks := ParseReadyOutput(output)
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestParseReadyOutput_SkipsMalformedLines(t *testing.T) {
	output := `some random text
bosun-abc   2    task   Good task
not-enough-fields
   `

	tasks := ParseReadyOutput(output)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "bosun-abc" {
		t.Errorf("ID = %q, want %q", tasks[0].ID, "bosun-abc")
	}
}

func TestParseReadyOutput_NoTitle(t *testing.T) {
	output := "proj-123   1    bug"
	tasks := ParseReadyOutput(output)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "" {
		t.Errorf("Title = %q, want empty", tasks[0].Title)
	}
}

func TestPickHighest(t *testing.T) {
	tasks := []Task{
		{ID: "a", Priority: 2},
		{ID: "b", Priority: 0},
		{ID: "c", Priority: 1},
	}

	best := PickHighest(tasks)
	if best == nil {
		t.Fatal("expected non-nil result")
	}
	if best.ID != "b" {
		t.Errorf("PickHighest = %q, want %q", best.ID, "b")
	}
}

func TestPickHighest_Empty(t *testing.T) {
	best := PickHighest(nil)
	if best != nil {
		t.Errorf("expected nil for empty list, got %v", best)
	}
}

func TestPickHighest_Single(t *testing.T) {
	tasks := []Task{{ID: "only", Priority: 3}}
	best := PickHighest(tasks)
	if best.ID != "only" {
		t.Errorf("PickHighest = %q, want %q", best.ID, "only")
	}
}

func TestFilterByLabel_NoFilter(t *testing.T) {
	tasks := []Task{
		{ID: "a-1", Title: "foo"},
		{ID: "b-2", Title: "bar"},
	}
	result := FilterByLabel(tasks, "")
	if len(result) != 2 {
		t.Errorf("expected 2 tasks with empty filter, got %d", len(result))
	}
}

func TestFilterByLabel_MatchesTitle(t *testing.T) {
	tasks := []Task{
		{ID: "a-1", Title: "implement CLI"},
		{ID: "b-2", Title: "fix database"},
	}
	result := FilterByLabel(tasks, "CLI")
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if result[0].ID != "a-1" {
		t.Errorf("ID = %q, want %q", result[0].ID, "a-1")
	}
}

func TestFilterByLabel_MatchesID(t *testing.T) {
	tasks := []Task{
		{ID: "skiff-42", Title: "something"},
		{ID: "bosun-11", Title: "other"},
	}
	result := FilterByLabel(tasks, "skiff")
	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}
	if result[0].ID != "skiff-42" {
		t.Errorf("ID = %q, want %q", result[0].ID, "skiff-42")
	}
}

func TestParsePriority(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"1", 1},
		{"4", 4},
		{"P0", 0},
		{"P2", 2},
		{"p3", 3},
		{"5", -1},
		{"abc", -1},
		{"", -1},
		{"P5", -1},
	}

	for _, tt := range tests {
		got := parsePriority(tt.input)
		if got != tt.want {
			t.Errorf("parsePriority(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// mockRunner records commands and returns preset output.
type mockRunner struct {
	calls   []string
	outputs map[string]string
	errors  map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, key)
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return "", nil
}

func (m *mockRunner) Start(_ context.Context, name string, args ...string) (*exec.Cmd, error) {
	return nil, fmt.Errorf("Start not implemented in mock")
}

func TestClaimWithRunner_Success(t *testing.T) {
	runner := newMockRunner()
	runner.outputs["br ready"] = `ID          PRI  TYPE   TITLE
bosun-3hj   1    task   run command
bosun-138   2    task   CLI scaffold`

	cfg := &config.Config{}

	taskID, err := ClaimWithRunner(context.Background(), cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskID != "bosun-3hj" {
		t.Errorf("claimed = %q, want %q", taskID, "bosun-3hj")
	}

	// Should have called br update to claim.
	found := false
	for _, call := range runner.calls {
		if strings.Contains(call, "br update bosun-3hj --status=in_progress") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected br update call, got calls: %v", runner.calls)
	}
}

func TestClaimWithRunner_NoTasks(t *testing.T) {
	runner := newMockRunner()
	runner.outputs["br ready"] = ""

	cfg := &config.Config{}

	taskID, err := ClaimWithRunner(context.Background(), cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskID != "" {
		t.Errorf("expected empty task ID, got %q", taskID)
	}
}

func TestClaimWithRunner_BrReadyError(t *testing.T) {
	runner := newMockRunner()
	runner.errors["br ready"] = fmt.Errorf("br not found")

	cfg := &config.Config{}

	_, err := ClaimWithRunner(context.Background(), cfg, runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "br ready") {
		t.Errorf("error = %q, expected to contain 'br ready'", err.Error())
	}
}

func TestClaimWithRunner_WithTaskFilter(t *testing.T) {
	runner := newMockRunner()
	// When TASK_FILTER is set, br ready is called with --label flag.
	// The output is already filtered by beads, so bosun picks the highest priority.
	runner.outputs["br ready --label repo:skiff"] = `skiff-42   1    task   Implement scale command
skiff-99   2    task   Add tests`

	cfg := &config.Config{TaskFilter: "repo:skiff"}

	taskID, err := ClaimWithRunner(context.Background(), cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Picks highest priority (P1) task.
	if taskID != "skiff-42" {
		t.Errorf("claimed = %q, want %q", taskID, "skiff-42")
	}

	// Verify br ready was called with --label flag.
	found := false
	for _, call := range runner.calls {
		if call == "br ready --label repo:skiff" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'br ready --label repo:skiff' call, got: %v", runner.calls)
	}
}
