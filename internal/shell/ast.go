package shell

// Node is the common marker interface for every AST node. Concrete nodes are
// pointers so they can be freely shared and compared by type.
type Node any

// Word is a single lexical word argument. Text retains the original source
// (including any quotes or escapes); Start and End are the UTF-8 byte span
// into the command.
type Word struct {
	Text  string
	Start int
	End   int
}

// Assignment is a leading name=value variable assignment on a command.
type Assignment struct {
	Name  string
	Value Word
	Start int
	End   int
}

// Redirect is a shell redirection attached to a simple command.
// Operator is one of ">", ">>", or "<" — the L0-supported set only.
type Redirect struct {
	Operator string
	Target   Word
	Start    int
	End      int
}

type Sequence struct {
	Items []Node
	Start int
	End   int
}

// Binary is left-associative so a && b || c lowers as (a && b) || c, matching
// SplitStages dependency wiring.
type Binary struct {
	Op    string
	Left  Node
	Right Node
	Start int
	End   int
}

// Pipeline is a '|'-separated sequence of commands that run in the same
// stage. Each entry keeps its own source span.
type Pipeline struct {
	Commands []Node
	Start    int
	End      int
}

type SimpleCommand struct {
	Assignments []Assignment
	Words       []Word
	Redirects   []Redirect
	Start       int
	End         int
}

type Subshell struct {
	Body  Node
	Start int
	End   int
}

// CommandSubstitution models a $(...) substitution. It is carried as a raw
// text node; command-substitution semantics are recorded as unknown by a
// later analysis stage.
type CommandSubstitution struct {
	Raw   string
	Start int
	End   int
}
