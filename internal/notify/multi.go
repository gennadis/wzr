package notify

import (
	"context"
	"errors"
)

// MultiNotifier fans out events to all registered notifiers.
// It continues calling all notifiers even if one fails, and returns
// a joined error if any failed.
type MultiNotifier struct {
	notifiers []Notifier
}

// NewMultiNotifier creates a MultiNotifier wrapping the given notifiers.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

// Notify calls all notifiers and returns a joined error of any failures.
func (m *MultiNotifier) Notify(ctx context.Context, event StepEvent) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
