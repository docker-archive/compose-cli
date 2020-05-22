package progress

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/buger/goterm"
	"github.com/morikuni/aec"
)

type spinner struct {
	time  time.Time
	index int
	chars []string
	stop  bool
}

func newSpinner() *spinner {
	return &spinner{
		index: 0,
		time:  time.Now(),
		chars: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
	}
}

func (s *spinner) String() string {
	if s.stop {
		return "⠿"
	}
	d := time.Since(s.time)
	if d.Milliseconds() > 100 {
		s.index = (s.index + 1) % len(s.chars)
	}
	return s.chars[s.index]
}

func (s *spinner) Stop() {
	s.stop = true
}

// EventStatus indicates the status of an action
type EventStatus int

const (
	// Working means that the current task is working
	Working EventStatus = iota
	// Done means that the current task is done
	Done
	// Error means that the current task has errored
	Error
)

// Event reprensents a progress event
type Event struct {
	ID         string
	Text       string
	Status     EventStatus
	StatusText string
	Done       bool
	startTime  time.Time
	endTime    time.Time
	spinner    *spinner
}

func (e *Event) stop() {
	e.endTime = time.Now()
	e.spinner.Stop()
}

// Writer can write multiple progress events
type Writer interface {
	Start(context.Context, <-chan Event) error
}

type writer struct {
	out      io.Writer
	events   map[string]Event
	eventIDs []string
	repeated bool
	numLines int
}

// NewWriter returns a new multi-progress writer
func NewWriter(out io.Writer) Writer {
	return &writer{
		out:      out,
		eventIDs: []string{},
		events:   map[string]Event{},
		repeated: false,
	}
}

func (w *writer) Start(ctx context.Context, ch <-chan Event) error {
	ticker := time.NewTicker(100 * time.Millisecond)

	done := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		case e, ok := <-ch:
			if !ok {
				done = true
			} else {
				if !contains(w.eventIDs, e.ID) {
					w.eventIDs = append(w.eventIDs, e.ID)
				}
				if _, ok := w.events[e.ID]; ok {
					event := w.events[e.ID]
					if event.Status != Done && e.Status == Done {
						event.stop()
					}
					event.Status = e.Status
					event.Text = e.Text
					event.StatusText = e.StatusText
					w.events[e.ID] = event
				} else {
					e.startTime = time.Now()
					e.spinner = newSpinner()
					w.events[e.ID] = e
				}
			}
		}

		if done {
			w.print()
			return nil
		}
		w.print()
	}
}

func (w *writer) print() {
	wd := goterm.Width()
	b := aec.EmptyBuilder
	for i := 0; i <= w.numLines; i++ {
		b = b.Up(1)
	}
	if !w.repeated {
		b = b.Down(1)
	}
	w.repeated = true
	fmt.Fprint(w.out, b.Column(0).ANSI)

	// Hide the cursor while we are printing
	fmt.Fprint(w.out, aec.Hide)
	defer fmt.Fprint(w.out, aec.Show)

	firstLine := fmt.Sprintf("[+] Running %d/%d", numDone(w.events), w.numLines)
	if w.numLines != 0 && numDone(w.events) == w.numLines {
		firstLine = aec.Apply(firstLine, aec.BlueF)
	}
	fmt.Fprintln(w.out, firstLine)

	var width int
	for _, v := range w.eventIDs {
		l := len(fmt.Sprintf("%s %s", w.events[v].ID, w.events[v].Text))
		if width < l {
			width = l
		}
	}

	lines := 0
	for _, v := range w.eventIDs {
		event := w.events[v]

		endTime := time.Now()
		if event.Status != Working {
			endTime = event.endTime
		}

		elapsed := endTime.Sub(event.startTime).Seconds()

		text := fmt.Sprintf(" %s %s %s%s %s",
			event.spinner.String(),
			event.ID,
			event.Text,
			strings.Repeat(" ", width-len(fmt.Sprintf("%s %s", w.events[v].ID, w.events[v].Text))),
			event.StatusText,
		)
		timer := fmt.Sprintf("%.1fs\n", elapsed)
		o := align(text, timer, wd)

		color := aec.WhiteF
		if event.Status == Done {
			color = aec.BlueF
		}
		if event.Status == Error {
			color = aec.RedF
		}
		o = aec.Apply(o, color)

		// nolint: errcheck
		fmt.Fprint(w.out, o)
		lines++
	}
	w.numLines = lines
}

func numDone(events map[string]Event) int {
	i := 0
	for _, e := range events {
		if e.Status == Done {
			i++
		}
	}
	return i
}

func align(l, r string, w int) string {
	return fmt.Sprintf("%-[2]*[1]s %[3]s", l, w-len(r)-1, r)
}

func contains(ar []string, needle string) bool {
	for _, v := range ar {
		if needle == v {
			return true
		}
	}
	return false
}
