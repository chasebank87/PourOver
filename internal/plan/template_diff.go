package plan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

const maxTemplateDiffBytes = 4096

// BuildTemplatePlan computes template_write actions from discovered template status.
// Missing and content-differ both emit template_write with a unified diff in Value.
// Same is a noop. Blocked (unexpected target type) errors unless replaceMode is backup.
func BuildTemplatePlan(statuses []discovery.TemplateStatus, replaceMode config.FileReplaceMode) (Plan, error) {
	var actions []Action
	for _, st := range statuses {
		switch st.Kind {
		case discovery.TemplateStatusMissing, discovery.TemplateStatusDiffer:
			actions = append(actions, Action{
				Type:   ActionTemplateWrite,
				Name:   st.File.Target,
				Source: st.File.Source,
				Value:  truncateDiff(unifiedDiff(st.File.Target, st.Current, st.Rendered)),
			})
		case discovery.TemplateStatusSame:
			continue
		case discovery.TemplateStatusBlocked:
			if replaceMode == config.FileReplaceBackup {
				note := "target is not a replaceable file; will backup then write rendered content"
				actions = append(actions, Action{
					Type:   ActionTemplateWrite,
					Name:   st.File.Target,
					Source: st.File.Source,
					Kind:   "backup",
					Value:  note,
				})
				continue
			}
			return Plan{}, fmt.Errorf(
				"target %q exists and is not a replaceable file (source %q)",
				st.File.Target, st.File.Source,
			)
		default:
			return Plan{}, fmt.Errorf("unknown template status %q for target %q", st.Kind, st.File.Target)
		}
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return Plan{Actions: actions}, nil
}

func truncateDiff(diff string) string {
	if len(diff) <= maxTemplateDiffBytes {
		return diff
	}
	return diff[:maxTemplateDiffBytes] + "\n... (diff truncated)"
}

// unifiedDiff returns a simple unified diff of old → new for plan JSON Value.
func unifiedDiff(path, oldContent, newContent string) string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	ops := lcsDiff(oldLines, newLines)

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ rendered\n", path)
	b.WriteString("@@\n")
	for _, op := range ops {
		switch op.kind {
		case diffKeep:
			b.WriteByte(' ')
			b.WriteString(op.line)
			b.WriteByte('\n')
		case diffRemove:
			b.WriteByte('-')
			b.WriteString(op.line)
			b.WriteByte('\n')
		case diffAdd:
			b.WriteByte('+')
			b.WriteString(op.line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type diffOpKind int

const (
	diffKeep diffOpKind = iota
	diffRemove
	diffAdd
)

type diffOp struct {
	kind diffOpKind
	line string
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Keep trailing empty line semantics: strings.Split("a\n", "\n") => ["a", ""]
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, diffOp{kind: diffKeep, line: a[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{kind: diffRemove, line: a[i]})
			i++
		} else {
			ops = append(ops, diffOp{kind: diffAdd, line: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{kind: diffRemove, line: a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{kind: diffAdd, line: b[j]})
	}
	return ops
}
