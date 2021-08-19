package dotutil

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var drawPatternCanvas *ebiten.Image

func init() {
	drawPatternCanvas = ebiten.NewImage(100, 100)
}

type PatternPosition int

const (
	PatternPositionTopLeft PatternPosition = iota
	PatternPositionCenter
)

type DrawPatternOption struct {
	Color        color.Color
	DotSize      float64
	DotInterval  float64
	Rotate       float64
	BasePosition PatternPosition
}

func DrawPattern(dst *ebiten.Image, pattern [][]int, x, y float64, option *DrawPatternOption) {
	var opt DrawPatternOption
	if option != nil {
		opt = *option
	}
	if opt.Color == nil {
		opt.Color = color.White
	}
	if opt.DotSize == 0 {
		opt.DotSize = 15
	}

	canvasWidth := int(float64(len(pattern[0]))*(opt.DotSize+opt.DotInterval) - opt.DotInterval)
	canvasHeight := int(float64(len(pattern))*(opt.DotSize+opt.DotInterval) - opt.DotInterval)
	if w, h := drawPatternCanvas.Size(); w < canvasWidth || h < canvasHeight {
		drawPatternCanvas = ebiten.NewImage(canvasWidth*2, canvasHeight*2)
	}
	canvas := drawPatternCanvas.SubImage(image.Rect(0, 0, canvasWidth, canvasHeight)).(*ebiten.Image)
	canvas.Clear()

	dotImg := emptyImage.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image)
	dotImg.Fill(opt.Color)

	for i := 0; i < len(pattern); i++ {
		for j := 0; j < len(pattern[i]); j++ {
			if pattern[i][j] != 0 {
				xij := float64(j) * (opt.DotSize + opt.DotInterval)
				yij := float64(i) * (opt.DotSize + opt.DotInterval)

				o := &ebiten.DrawImageOptions{}
				o.GeoM.Scale(opt.DotSize, opt.DotSize)
				o.GeoM.Translate(xij, yij)

				canvas.DrawImage(dotImg, o)
			}
		}
	}

	o := &ebiten.DrawImageOptions{}
	o.GeoM.Rotate(opt.Rotate)
	o.GeoM.Translate(x, y)

	switch opt.BasePosition {
	case PatternPositionCenter:
		r := math.Sqrt(math.Pow(float64(canvasWidth), 2)+math.Pow(float64(canvasHeight), 2)) / 2
		rad := math.Atan2(float64(canvasHeight), float64(canvasWidth))
		o.GeoM.Translate(-r*math.Cos(opt.Rotate+rad), -r*math.Sin(opt.Rotate+rad))
	}

	dst.DrawImage(canvas, o)
}
