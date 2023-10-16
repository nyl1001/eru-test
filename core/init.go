package core

import (
	"context"
)

func Prepare(ctx context.Context) error {
	return Init(ctx)
}
