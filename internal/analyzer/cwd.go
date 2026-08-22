package analyzer

import (
	"strconv"
	"strings"

	"github.com/phaethix/runmark/internal/effect"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/logicalpath"
	"github.com/phaethix/runmark/internal/shell"
)

// resolveStageCWDs computes the logical working directory each stage runs in,
// the way one shell session would: a cd at depth d moves later stages at the
// same depth, a subshell starts from a copy of its parent's directory and
// never leaks changes out, and a stage gated on a cd's failure sees the
// pre-cd directory. Stage order in the flat list is source order, which is
// the only order a static analysis can defend.
//
// It returns one cwd per stage plus non-blocking unknowns for runtime-
// dependent directory changes (bare cd, cd -, dynamic operands) and for
// stages reachable from both a success and a failure timeline of a cd.
func resolveStageCWDs(root string, stages []shell.Stage, env map[string]string) ([]string, []ir.Unknown) {
	var unknowns []ir.Unknown
	cwds := make([]string, len(stages))
	stack := []string{root}

	// Direct on-failure dependents of stage i keep the pre-cd directory: the
	// cd only moved the session when it succeeded, which is exactly the
	// branch they do not run in. Dependencies always point back, so the cd
	// has been processed before any dependent is reached.
	failureDependents := map[int][]int{}
	for j, st := range stages {
		if st.Condition.Kind != shell.ConditionOnFailure {
			continue
		}
		dep := st.Condition.DependsOn - 1
		failureDependents[dep] = append(failureDependents[dep], j)
	}

	failureOverrides := map[int]string{}
	anyCDWithFailureBranch := false
	for i, st := range stages {
		for len(stack) > st.Depth+1 {
			stack = stack[:len(stack)-1]
		}
		for len(stack) < st.Depth+1 {
			stack = append(stack, stack[len(stack)-1])
		}

		effective := stack[len(stack)-1]
		if pre, ok := failureOverrides[i]; ok {
			effective = pre
		}
		cwds[i] = effective

		cmd := soleSimpleCommand(st)
		op := cdOperand(cmd)
		if op == nil {
			continue
		}

		pre := stack[len(stack)-1]
		if operand, dynamic := resolveCDOperand(op.Text, env); !dynamic {
			next, _ := logicalpath.NormalizeLogicalPath(operand, pre)
			stack[len(stack)-1] = next
		} else {
			unknowns = append(unknowns, ir.Unknown{
				Code:     ir.UnknownCwdRuntimeDependent,
				Scope:    "stage:" + strconv.Itoa(i),
				Message:  "cd operand is runtime-dependent; later stages stay on the request cwd",
				Evidence: []ir.Evidence{CommandEvidence(op.Start, op.End, op.Text)},
				Blocking: false,
			})
		}
		if deps := failureDependents[i]; len(deps) > 0 {
			anyCDWithFailureBranch = true
			for _, j := range deps {
				failureOverrides[j] = pre
			}
		}
	}

	// A cd with a failure branch leaves later unconditional stages on two
	// possible timelines; one report-scoped unknown is more honest than
	// picking a side.
	if anyCDWithFailureBranch {
		unknowns = append(unknowns, ir.Unknown{
			Code:     ir.UnknownCwdRuntimeDependent,
			Scope:    "report",
			Message:  "a cd has a failure branch, so later stages may run on either directory timeline",
			Evidence: []ir.Evidence{},
			Blocking: false,
		})
	}
	return cwds, unknowns
}

func soleSimpleCommand(st shell.Stage) *shell.SimpleCommand {
	if len(st.Commands) != 1 {
		return nil
	}
	cmd, ok := st.Commands[0].(*shell.SimpleCommand)
	if !ok || len(cmd.Words) == 0 {
		return nil
	}
	return cmd
}

// cdOperand returns the directory operand of a cd stage: -L/-P/-- flags are
// dropped. A bare cd (no operand) also returns a non-nil word so callers
// can flag it as runtime-dependent (it goes to $HOME). Returns nil only for
// non-cd commands.
func cdOperand(cmd *shell.SimpleCommand) *shell.Word {
	if cmd == nil {
		return nil
	}
	name := strings.ReplaceAll(cmd.Words[0].Text, `\`, "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if name != "cd" {
		return nil
	}
	var operands []shell.Word
	for _, w := range cmd.Words[1:] {
		if len(operands) == 0 && strings.HasPrefix(w.Text, "-") && w.Text != "-" {
			continue
		}
		operands = append(operands, w)
	}
	switch len(operands) {
	case 0:
		return &shell.Word{Text: "", Start: cmd.Words[0].End, End: cmd.Words[0].End}
	default:
		return &operands[0]
	}
}

// resolveCDOperand substitutes env-provided values so `cd "$SUB"` can move
// the logical cwd when the caller actually told us what SUB is. Anything
// still containing runtime constructs stays dynamic rather than guessed.
func resolveCDOperand(text string, env map[string]string) (string, bool) {
	// Bare cd goes to $HOME and cd - toggles $OLDPWD; both are shell runtime
	// state, so inventing either destination would fabricate a path.
	if text == "" || text == "-" {
		return "", true
	}
	unquoted := strings.Trim(text, "\"'")
	resolved := effect.SubstituteEnvRefs(unquoted, env)
	if resolved == "" {
		return "", true
	}
	if strings.ContainsAny(resolved, "$`") ||
		strings.ContainsAny(resolved, "*?[") ||
		isTildePath(resolved) {
		return "", true
	}
	return resolved, false
}
