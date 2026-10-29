// platform/notify/notify_other.go

//go:build !linux

package notify

import (
	"context"
	"errors"
)

// Desktop sends notifications through the platform notification service.
type Desktop struct{}

func New(string) *Desktop { return &Desktop{} }

func (*Desktop) Send(ctx context.Context, _, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.ErrUnsupported
}

func (d *Desktop) SendAction(ctx context.Context, title, body, _ string, _ func()) error {
	return d.Send(ctx, title, body)
}

func (*Desktop) Close() {}
