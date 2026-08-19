package facts

import (
	"fmt"
	"strings"
)

// FormatText is a short human summary of projected facts for CLI text output
// and hook additionalContext. It does not re-analyze — only pretty-prints f.
func FormatText(f RunmarkFacts) string {
	var b strings.Builder
	fmt.Fprintf(&b, "schema_version: %s\n", f.SchemaVersion)
	fmt.Fprintf(&b, "unknown: %v\n", f.Unknown)
	if len(f.UnknownReasons) > 0 {
		fmt.Fprintf(&b, "unknown_reasons: %s\n", strings.Join(f.UnknownReasons, ", "))
	}
	writeTouchLines(&b, "read", f.Touches.Read)
	writeTouchLines(&b, "write", f.Touches.Write)
	writeTouchLines(&b, "delete", f.Touches.Delete)
	fmt.Fprintf(&b, "boundary: outside_workspace=%v sensitive_path=%v destructive=%v external_network=%v opaque_script=%v\n",
		f.Boundary.OutsideWorkspace,
		f.Boundary.SensitivePath,
		f.Boundary.Destructive,
		f.Boundary.ExternalNetwork,
		f.Boundary.OpaqueScript,
	)
	if len(f.Scripts) > 0 {
		fmt.Fprintln(&b, "scripts:")
		for _, s := range f.Scripts {
			fmt.Fprintf(&b, "  - tool=%s name=%s source=%s\n", s.Tool, s.Name, s.Source)
		}
	}
	if len(f.Evidence) > 0 {
		fmt.Fprintln(&b, "evidence:")
		for _, ev := range f.Evidence {
			label := ev.Source
			if ev.Path != "" {
				label = label + ":" + ev.Path
			}
			if ev.Field != "" {
				label = label + "#" + ev.Field
			}
			fmt.Fprintf(&b, "  - %s: %s\n", label, ev.Snippet)
		}
	}
	return b.String()
}

func writeTouchLines(b *strings.Builder, kind string, paths []string) {
	if len(paths) == 0 {
		fmt.Fprintf(b, "%s: (none)\n", kind)
		return
	}
	fmt.Fprintf(b, "%s:\n", kind)
	for _, p := range paths {
		fmt.Fprintf(b, "  - %s\n", p)
	}
}
