package shell

// ConditionKind selects the gating semantics of a stage. It is intentionally
// closed to exactly these three values so that a stage's condition can be
// lowered 1:1 onto the IR-level ConditionKind vocabulary without translation.
type ConditionKind string

const (
	ConditionAlways    ConditionKind = "always"
	ConditionOnSuccess ConditionKind = "on_success"
	ConditionOnFailure ConditionKind = "on_failure"
)

// Condition describes how a stage is gated and which earlier stage it refers
// to. It always references a previously emitted stage, so a consumer can
// resolve dependencies by looking back over the already-collected stages.
type Condition struct {
	Kind ConditionKind

	// DependsOn names the 1-based stage index this condition gates on. It is
	// always 0 for ConditionAlways, because an unconditional stage has no
	// dependency to point at.
	DependsOn int
}

// Stage is one executable unit in the flattened stage graph. Several commands
// share a Stage when they must run untouched by conditionals (a pipeline), and
// Index is globally consecutive across nesting so downstream consumers can
// treat stage identity as a dense, stable ordinal.
type Stage struct {
	Index     int
	Commands  []Node
	Condition Condition

	// Depth records subshell nesting (0 = top level). A cd inside a subshell
	// must not move the working directory of stages after the subshell, so cwd
	// tracking needs this marker even though the stage list itself is flat.
	Depth int
}

// SplitStages flattens a parsed AST into an ordered, globally consecutive
// list of stages. The ordering and gating mirror shell control precedence:
//
//   - a pipeline binds its commands into one stage (they must all run);
//   - ';' separates stages that never gate on each other;
//   - && / || gate the right-hand side on success / failure of everything left
//     of the operator, and index is global, so a parenthesized subshell keeps
//     expanding its body over the same monotone counter.
//
// Stages are emitted left-to-right in source order, and the numbering reflects
// that order, not any notion of dependency depth.
func SplitStages(root Node) []Stage {
	b := &stageBuilder{}
	b.explode(root, condition(ConditionAlways))
	return b.stages
}

type stageBuilder struct {
	stages []Stage
	depth  int
}

func (b *stageBuilder) add(commands []Node, c Condition) int {
	idx := len(b.stages) + 1
	b.stages = append(b.stages, Stage{
		Index:     idx,
		Commands:  commands,
		Condition: c,
		Depth:     b.depth,
	})
	return idx
}

// explode emits stages for node in source order and returns the last index it
// produced. c is the condition inherited from an enclosing && / || right-hand
// side; a standalone command defaults to always. The default arm returns 0 for
// node kinds that are outside the L0 grammar, because they must not be turned
// into a stage yet.
func (b *stageBuilder) explode(node Node, c Condition) int {
	switch n := node.(type) {
	case nil:
		return 0
	case *Subshell:
		b.depth++
		last := b.explode(n.Body, c)
		b.depth--
		return last
	case *SimpleCommand:
		return b.add([]Node{n}, c)
	case *CommandSubstitution:
		return b.add([]Node{n}, c)
	case *Pipeline:
		return b.add(n.Commands, c)
	case *Sequence:
		last := 0
		for _, item := range n.Items {
			last = max(last, b.explode(item, c))
		}
		return last
	case *Binary:
		left := b.explode(n.Left, c)
		var depKind ConditionKind
		switch n.Op {
		case "&&":
			depKind = ConditionOnSuccess
		case "||":
			depKind = ConditionOnFailure
		}
		return b.explode(n.Right, Condition{Kind: depKind, DependsOn: left})
	default:
		return 0
	}
}

func condition(k ConditionKind) Condition {
	return Condition{Kind: k}
}
