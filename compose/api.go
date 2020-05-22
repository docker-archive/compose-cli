package compose

import (
	"context"

	"github.com/docker/api/progress"
)

// Service manages a compose project
type Service interface {
	// Up executes the equivalent to a `compose up`
	Up(ctx context.Context, opts ProjectOptions, channel chan<- progress.Event) error
	// Down executes the equivalent to a `compose down`
	Down(ctx context.Context, opts ProjectOptions) error
}
