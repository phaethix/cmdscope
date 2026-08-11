package analyzer

import "github.com/phaethix/cmdscope/internal/ir"

// CommandEvidence builds command-sourced evidence with a half-open [start,end)
// span. Invalid spans drop both pointers so ValidateReport never sees a
// one-sided interval; the snippet alone still anchors the claim.
func CommandEvidence(start, end int, snippet string) ir.Evidence {
	ev := ir.Evidence{Source: ir.EvidenceCommand, Snippet: snippet}
	if start >= 0 && end > start {
		ev.StartByte = intPtr(start)
		ev.EndByte = intPtr(end)
	}
	return ev
}

// FileEvidence locates proof in caller-supplied context or an embedded script.
// Spans are omitted because those sources are addressed by path/field, not by
// command byte offsets.
func FileEvidence(source ir.EvidenceSource, path, field, snippet string) ir.Evidence {
	return ir.Evidence{
		Source:  source,
		Path:    path,
		Field:   field,
		Snippet: snippet,
	}
}

// EnsureEffectHasEvidence guarantees a non-nil, non-empty Evidence slice.
// Extractors can forget the invariant; filling a minimal command snippet from
// RawTarget (else Target) keeps ValidateReport from rejecting otherwise-valid
// effects for a missing trail.
func EnsureEffectHasEvidence(ef *ir.Effect) {
	if ef == nil || len(ef.Evidence) > 0 {
		return
	}
	snippet := ef.RawTarget
	if snippet == "" {
		snippet = ef.Target
	}
	ef.Evidence = []ir.Evidence{CommandEvidence(0, 0, snippet)}
}

func intPtr(n int) *int { return &n }
