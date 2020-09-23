package formatter

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func PrintPrettySection(out io.Writer, printer func(writer io.Writer), headers ...string) error {
	w := tabwriter.NewWriter(out, 20, 1, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	printer(w)
	return w.Flush()
}
