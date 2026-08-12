package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/phaethix/runmark/internal/adapter/codex"
	"github.com/phaethix/runmark/internal/analyzer"
	"github.com/phaethix/runmark/internal/facts"
	"github.com/phaethix/runmark/internal/ir"
	"github.com/phaethix/runmark/internal/render"
)

const (
	exitOK                = 0
	exitInternal          = 1
	exitUsage             = 2
	exitContractViolation = 3
)

// Run dispatches CLI subcommands. args are os.Args[1:] (no program name).
// Exit codes: 0 ok, 2 usage/input, 3 internal contract, 1 other I/O.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			usage(stderr)
			return exitUsage
		}
		if err := PrintVersion(stdout); err != nil {
			fmt.Fprintln(stderr, "runmark: write version:", err)
			return exitInternal
		}
		return exitOK
	case "analyze":
		return runAnalyze(args[1:], stdout, stderr)
	case "hook":
		return runHook(args[1:], stdout, stderr)
	default:
		usage(stderr)
		return exitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: runmark version")
	fmt.Fprintln(w, "       runmark analyze <command> [--cwd path] [--context-file file] [--format facts|impact|text]")
	fmt.Fprintln(w, "       runmark hook codex")
}

func runHook(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "codex" {
		fmt.Fprintln(stderr, "runmark hook: expected \"codex\"")
		usage(stderr)
		return exitUsage
	}
	return codex.Handle(context.Background(), os.Stdin, stdout, stderr)
}

func runAnalyze(args []string, stdout, stderr io.Writer) int {
	opts, err := parseAnalyzeArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, "runmark analyze:", err)
		usage(stderr)
		return exitUsage
	}

	switch opts.format {
	case "facts", "impact", "text":
	default:
		fmt.Fprintf(stderr, "runmark analyze: unknown format %q\n", opts.format)
		return exitUsage
	}

	ctx, err := loadContext(opts.contextFile, opts.cwd)
	if err != nil {
		return writeInputError(stderr, err)
	}

	report, err := analyzer.Analyze(context.Background(), ir.AnalyzeRequest{
		Command: opts.command,
		Context: ctx,
	})
	if err != nil {
		return writeInputError(stderr, err)
	}

	if err := render.Validate(report); err != nil {
		return writeContractError(stderr, err)
	}

	switch opts.format {
	case "impact":
		body, err := render.JSON(report)
		if err != nil {
			return writeContractError(stderr, err)
		}
		if _, err := stdout.Write(append(body, '\n')); err != nil {
			fmt.Fprintln(stderr, "runmark: write output:", err)
			return exitInternal
		}
		return exitOK
	case "facts", "text":
		f := facts.Project(report)
		if err := facts.Validate(f); err != nil {
			fmt.Fprintln(stderr, "runmark: facts contract:", err)
			return exitContractViolation
		}
		if opts.format == "facts" {
			body, err := json.Marshal(f)
			if err != nil {
				fmt.Fprintln(stderr, "runmark: encode facts:", err)
				return exitInternal
			}
			if _, err := stdout.Write(append(body, '\n')); err != nil {
				fmt.Fprintln(stderr, "runmark: write output:", err)
				return exitInternal
			}
			return exitOK
		}
		if _, err := io.WriteString(stdout, facts.FormatText(f)); err != nil {
			fmt.Fprintln(stderr, "runmark: write output:", err)
			return exitInternal
		}
		return exitOK
	default:
		return exitUsage
	}
}

type analyzeOpts struct {
	command     string
	cwd         string
	contextFile string
	format      string
}

// parseAnalyzeArgs accepts flags before or after the single command argv so
// `runmark analyze '<cmd>' --cwd …` matches the product contract (stdlib flag
// stops at the first non-flag and cannot express that order alone).
func parseAnalyzeArgs(args []string) (analyzeOpts, error) {
	opts := analyzeOpts{format: "facts"}
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			positional = append(positional, args[i+1:]...)
			i = len(args)
		case a == "-h", a == "--help":
			return analyzeOpts{}, errors.New("help requested")
		case a == "--cwd":
			if i+1 >= len(args) {
				return analyzeOpts{}, errors.New("--cwd requires a value")
			}
			i++
			opts.cwd = args[i]
		case strings.HasPrefix(a, "--cwd="):
			opts.cwd = strings.TrimPrefix(a, "--cwd=")
		case a == "--context-file":
			if i+1 >= len(args) {
				return analyzeOpts{}, errors.New("--context-file requires a value")
			}
			i++
			opts.contextFile = args[i]
		case strings.HasPrefix(a, "--context-file="):
			opts.contextFile = strings.TrimPrefix(a, "--context-file=")
		case a == "--format":
			if i+1 >= len(args) {
				return analyzeOpts{}, errors.New("--format requires a value")
			}
			i++
			opts.format = args[i]
		case strings.HasPrefix(a, "--format="):
			opts.format = strings.TrimPrefix(a, "--format=")
		case strings.HasPrefix(a, "-"):
			return analyzeOpts{}, fmt.Errorf("unknown flag %q", a)
		default:
			positional = append(positional, a)
		}
	}
	if len(positional) != 1 {
		return analyzeOpts{}, errors.New("exactly one <command> argument is required")
	}
	opts.command = positional[0]
	return opts, nil
}

// loadContext reads --context-file only in the app layer and applies --cwd
// override. Core analysis still sees an explicit AnalysisContext only.
func loadContext(contextFile, cwdFlag string) (*ir.AnalysisContext, error) {
	var ctx *ir.AnalysisContext
	if contextFile != "" {
		data, err := os.ReadFile(contextFile)
		if err != nil {
			return nil, ir.NewValidationError(ir.ErrCodeInvalidContextJSON, "cannot read context-file: "+err.Error())
		}
		parsed, err := ir.ParseAnalysisContextJSON(data)
		if err != nil {
			return nil, err
		}
		ctx = parsed
	}
	if cwdFlag != "" {
		if ctx == nil {
			ctx = &ir.AnalysisContext{CWD: cwdFlag, Files: map[string]string{}, Env: map[string]string{}}
		} else {
			ctx.CWD = cwdFlag
		}
	}
	return ctx, nil
}

func writeInputError(stderr io.Writer, err error) int {
	if ve, ok := errors.AsType[*ir.ValidationError](err); ok {
		body, encErr := json.Marshal(ve)
		if encErr != nil {
			fmt.Fprintln(stderr, ve.Error())
			return exitUsage
		}
		fmt.Fprintln(stderr, string(body))
		return exitUsage
	}
	fmt.Fprintln(stderr, "runmark:", err)
	return exitUsage
}

func writeContractError(stderr io.Writer, err error) int {
	if cv, ok := errors.AsType[*ir.ContractViolationError](err); ok {
		body, encErr := json.Marshal(cv)
		if encErr != nil {
			fmt.Fprintln(stderr, cv.Error())
			return exitContractViolation
		}
		fmt.Fprintln(stderr, string(body))
		return exitContractViolation
	}
	fmt.Fprintln(stderr, "runmark:", err)
	return exitContractViolation
}
