package dotutil

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type LineDotPosition int

const (
	LineDotPositionCenter LineDotPosition = iota
	LineDotPositionLeftSide
	LineDotPositionRightSide
)

var drawLineCanvas *ebiten.Image

func init() {
	drawLineCanvas = ebiten.NewImage(100, 100)
}

type DrawLineOption struct {
	Color       color.Color
	DotSize     float64
	Interval    float64
	DotPosition LineDotPosition
}

func DrawLine(dst *ebiten.Image, srcX, srcY, destX, destY float64, option *DrawLineOption) {
	var opt DrawLineOption
	if option != nil {
		opt = *option
	}
	if opt.Color == nil {
		opt.Color = color.White
	}
	if opt.DotSize == 0 {
		opt.DotSize = 15
	}
	if opt.Interval == 0 {
		opt.Interval = 2
	}

	lineLength := math.Sqrt(math.Pow(srcX-destX, 2) + math.Pow(srcY-destY, 2))
	angle := math.Atan2(destY-srcY, destX-srcX)

	dotImg := emptyImage.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image)
	dotImg.Fill(opt.Color)

	if w, h := drawLineCanvas.Size(); w < int(lineLength) || h < int(opt.DotSize) {
		drawLineCanvas = ebiten.NewImage(int(lineLength)*2, int(opt.DotSize)*2)
	}
	canvas := drawLineCanvas.SubImage(image.Rect(0, 0, int(lineLength), int(opt.DotSize))).(*ebiten.Image)
	canvas.Clear()

	var x float64 = 0
	for x <= lineLength {
		o := &ebiten.DrawImageOptions{}
		o.GeoM.Scale(opt.DotSize, opt.DotSize)
		o.GeoM.Translate(x, 0)
		canvas.DrawImage(dotImg, o)
		x += opt.DotSize + opt.Interval
	}

	o := &ebiten.DrawImageOptions{}
	switch opt.DotPosition {
	case LineDotPositionLeftSide:
		o.GeoM.Translate(0, -opt.DotSize)
	case LineDotPositionCenter:
		o.GeoM.Translate(0, -opt.DotSize/2)
	}
	if angle != 0 {
		o.GeoM.Rotate(angle)
	}
	o.GeoM.Translate(srcX, srcY)

	dst.DrawImage(canvas, o)
}
