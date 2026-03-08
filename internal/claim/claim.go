// Package claim handles picking the highest-priority ready task from beads.
package claim

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chronick/bosun/internal/config"
	"github.com/chronick/bosun/internal/toolexec"
)

// Task represents a parsed beads task from `br ready` output.
type Task struct {
	ID       string
	Priority int    // 0=critical .. 4=backlog
	Title    string
	Type     string
	Labels   string
}

// ParseReadyOutput parses the tabular output of `br ready`.
//
// Expected format (one task per line, tab or multi-space separated):
//
//	ID          PRI  TYPE   TITLE
//	bosun-3hj   1    task   run command - full lifecycle loop
//	bosun-138   2    task   CLI scaffold with cobra
//
// Lines that don't match the expected format are skipped (headers, empty lines, etc).
func ParseReadyOutput(output string) []Task {
	var tasks []Task
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines (contain "ID" or dashes or are clearly headers).
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "id") || strings.HasPrefix(line, "─") || strings.HasPrefix(line, "-") {
			continue
		}

		task := parseLine(line)
		if task.ID != "" {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// parseLine extracts a Task from a single line of `br ready` output.
// It handles both tab-separated and multi-space-separated formats.
func parseLine(line string) Task {
	// Split on whitespace, then reconstruct the title from remaining fields.
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return Task{}
	}

	// Field 0: ID (must contain a dash like "bosun-3hj")
	id := fields[0]
	if !strings.Contains(id, "-") {
		return Task{}
	}

	// Field 1: Priority (single digit 0-4, or P0-P4)
	priStr := fields[1]
	pri := parsePriority(priStr)
	if pri < 0 {
		return Task{}
	}

	// Field 2: Type (task, bug, feature, epic, chore, docs, question)
	taskType := fields[2]

	// Remaining fields: title
	title := ""
	if len(fields) > 3 {
		title = strings.Join(fields[3:], " ")
	}

	return Task{
		ID:       id,
		Priority: pri,
		Title:    title,
		Type:     taskType,
	}
}

// parsePriority parses "0", "1", "P0", "P1", etc. Returns -1 on failure.
func parsePriority(s string) int {
	s = strings.TrimPrefix(strings.ToUpper(s), "P")
	if len(s) == 1 && s[0] >= '0' && s[0] <= '4' {
		return int(s[0] - '0')
	}
	return -1
}

// PickHighest returns the highest-priority (lowest number) task from the list.
// Returns nil if the list is empty.
func PickHighest(tasks []Task) *Task {
	if len(tasks) == 0 {
		return nil
	}
	best := &tasks[0]
	for i := 1; i < len(tasks); i++ {
		if tasks[i].Priority < best.Priority {
			best = &tasks[i]
		}
	}
	return best
}

// FilterByLabel filters tasks by a label substring (from TASK_FILTER).
// If filter is empty, returns all tasks.
func FilterByLabel(tasks []Task, filter string) []Task {
	if filter == "" {
		return tasks
	}
	var result []Task
	for _, t := range tasks {
		// Match against ID, title, type, or labels.
		if strings.Contains(t.ID, filter) ||
			strings.Contains(strings.ToLower(t.Title), strings.ToLower(filter)) ||
			strings.Contains(t.Type, filter) ||
			strings.Contains(t.Labels, filter) {
			result = append(result, t)
		}
	}
	return result
}

// Claim picks and claims the highest-priority ready task.
// Returns the task ID, or "" if no tasks are available.
func Claim(cfg *config.Config) (string, error) {
	return ClaimWithRunner(context.Background(), cfg, &toolexec.DefaultRunner{})
}

// ClaimWithRunner picks and claims a task using the provided runner.
func ClaimWithRunner(ctx context.Context, cfg *config.Config, runner toolexec.Runner) (string, error) {
	// Run `br ready` to get available tasks.
	// If TASK_FILTER is set, pass it to br as a --label flag so beads
	// does the filtering server-side. No need to double-filter locally.
	args := []string{"ready"}
	if cfg.TaskFilter != "" {
		args = append(args, "--label", cfg.TaskFilter)
	}

	output, err := toolexec.BR(ctx, runner, args...)
	if err != nil {
		return "", fmt.Errorf("br ready: %w", err)
	}

	tasks := ParseReadyOutput(output)
	best := PickHighest(tasks)

	if best == nil {
		log.Println("bosun: no ready tasks available")
		return "", nil
	}

	// Claim the task by setting status to in_progress.
	_, err = toolexec.BR(ctx, runner, "update", best.ID, "--status=in_progress")
	if err != nil {
		return "", fmt.Errorf("br update %s: %w", best.ID, err)
	}

	log.Printf("bosun: claimed task %s (P%d): %s", best.ID, best.Priority, best.Title)
	return best.ID, nil
}
