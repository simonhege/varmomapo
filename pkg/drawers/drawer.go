package drawers

import (
	"context"
	"image"
)

type Drawer interface {
	Draw(ctx context.Context, x, y, level int) (image.Image, error)
}