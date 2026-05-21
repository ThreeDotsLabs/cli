package diagnostics

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"

	"github.com/ThreeDotsLabs/cli/internal"
)

const stepLabelWidth = 32

// renderer prints diagnostic progress and the final report. All output goes to
// stderr so stdout stays clean for piping. On a TTY it overwrites the in-flight
// "...running" line on completion; on non-TTY it prints append-only so the
// output is clean when redirected to a file for a bug report.
type renderer struct {
	w   io.Writer
	tty bool
}

func newRenderer() *renderer {
	w := os.Stderr
	return &renderer{
		w:   w,
		tty: term.IsTerminal(int(w.Fd())),
	}
}

func (r *renderer) sep() string {
	return color.HiBlackString(strings.Repeat("─", internal.TerminalWidth()))
}

func (r *renderer) printHeader() {
	fmt.Fprintln(r.w, r.sep())
	fmt.Fprintln(r.w, color.New(color.Bold).Sprint("tdl diagnostics"))
	fmt.Fprintln(r.w, color.HiBlackString("Tests connectivity to the verification server, layer by layer."))
	fmt.Fprintln(r.w, r.sep())
}

func (r *renderer) printSnapshot(s Snapshot) {
	dim := color.HiBlackString
	line := func(k, v string) {
		fmt.Fprintf(r.w, "  %s %s\n", dim(fmt.Sprintf("%-18s", k+":")), v)
	}
	fmt.Fprintln(r.w, color.New(color.Bold).Sprint("Environment"))
	line("CLI version", fmt.Sprintf("%s (%s)", s.CLIVersion, s.CLICommit))
	line("OS / arch", fmt.Sprintf("%s / %s", s.OS, s.Arch))
	line("Go version", s.GoVersion)
	line("Configured", fmt.Sprintf("%t", s.Configured))
	line("Server addr", s.Server)
	line("Region", emptyDash(s.Region))
	line("Effective target", s.Target)
	line("Insecure TLS", fmt.Sprintf("%t", s.Insecure))
	line("Token", s.TokenStatus)
	if len(s.ProxyEnv) > 0 {
		line("Proxy env", color.YellowString("set (see check below)"))
	}
	if len(s.Tunnels) > 0 {
		line("VPN / tunnels", color.YellowString("active: "+strings.Join(s.Tunnels, ", ")))
	}
	fmt.Fprintln(r.w, r.sep())
	fmt.Fprintln(r.w, color.New(color.Bold).Sprint("Checks"))
}

func emptyDash(s string) string {
	if s == "" {
		return color.HiBlackString("(none)")
	}
	return s
}

func (r *renderer) startStep(idx, total int, name string) {
	if !r.tty {
		return
	}
	fmt.Fprintf(r.w, "  [%*d/%d] %-*s %s",
		digitWidth(total), idx, total, stepLabelWidth, name,
		color.HiBlackString("..."),
	)
}

func (r *renderer) finishStep(idx, total int, res Result) {
	if r.tty {
		fmt.Fprint(r.w, "\r\033[2K")
	}
	icon := color.GreenString("✓")
	switch {
	case res.Skipped:
		icon = color.YellowString("•")
	case !res.Pass:
		icon = color.RedString("✗")
	}
	dur := color.HiBlackString(fmt.Sprintf("(%s)", res.Duration.Round(1_000_000))) // ms precision
	fmt.Fprintf(r.w, "  [%*d/%d] %-*s %s  %s %s\n",
		digitWidth(total), idx, total, stepLabelWidth, res.Name, icon, res.Detail, dur,
	)
	for _, s := range res.Subitems {
		fmt.Fprintf(r.w, "         %s\n", color.HiBlackString(s))
	}
}

func digitWidth(n int) int {
	if n < 10 {
		return 1
	}
	w := 0
	for n > 0 {
		w++
		n /= 10
	}
	return w
}

func (r *renderer) printSummary(results []Result) {
	fmt.Fprintln(r.w, r.sep())
	fmt.Fprintln(r.w, color.New(color.Bold).Sprint("Summary"))
	for _, res := range results {
		icon := color.GreenString("✓")
		switch {
		case res.Skipped:
			icon = color.YellowString("•")
		case !res.Pass:
			icon = color.RedString("✗")
		}
		fmt.Fprintf(r.w, "  %s  %s — %s\n", icon, res.Name, res.Detail)
	}
}

func (r *renderer) printInterpretation(text string) {
	fmt.Fprintln(r.w, r.sep())
	fmt.Fprintln(r.w, color.New(color.Bold).Sprint("Interpretation"))
	fmt.Fprintln(r.w, text)
	fmt.Fprintln(r.w, r.sep())
}
