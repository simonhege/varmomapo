package debugdrawer

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/simonhege/varmomapo/pkg/drawers"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

func New(tileSize int) drawers.Drawer {
	return drawer{
		tileSize: tileSize,
	}
}

type drawer struct {
	tileSize int
}

func (dr drawer) Draw(ctx context.Context, x, y, level int) (image.Image, error) {

	r := image.Rect(0, 0, dr.tileSize, dr.tileSize)
	img := image.NewRGBA(r)
	for i := 0; i < r.Max.X; i++ {
		for j := 0; j < r.Max.Y; j++ {
			img.Set(i, j, color.Black)
		}
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.White),
		Face: inconsolata.Regular8x16,
		Dot:  fixed.Point26_6{X: fixed.I(5), Y: fixed.I(5 + 16)},
	}
	d.DrawString(fmt.Sprintf("(%d|%d) lvl=%d", x, y, level))

	return img, nil
}
