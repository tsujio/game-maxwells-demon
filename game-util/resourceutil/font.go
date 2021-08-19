package resourceutil

import (
	"io/fs"
	"io/ioutil"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type LoadFontOption struct {
	DPI                              float64
	LargeSize, MediumSize, SmallSize float64
}

type Font struct {
	Face        font.Face
	FaceOptions *opentype.FaceOptions
}

func LoadFont(repository fs.FS, fontName string, option *LoadFontOption) (largeFont, mediumFont, smallFont *Font, err error) {
	var opt LoadFontOption
	if option != nil {
		opt = *option
	}
	if opt.DPI == 0 {
		opt.DPI = 72
	}
	if opt.MediumSize == 0 {
		opt.MediumSize = 24
	}
	if opt.LargeSize == 0 {
		opt.LargeSize = opt.MediumSize * 1.5
	}
	if opt.SmallSize == 0 {
		opt.SmallSize = opt.MediumSize / 2
	}

	f, err := repository.Open(fontName)
	if err != nil {
		return
	}
	defer f.Close()
	fontData, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	tt, err := opentype.Parse(fontData)
	if err != nil {
		return
	}

	largeFaceOptions := &opentype.FaceOptions{
		Size:    opt.LargeSize,
		DPI:     opt.DPI,
		Hinting: font.HintingFull,
	}
	largeFace, err := opentype.NewFace(tt, largeFaceOptions)
	if err != nil {
		return
	}
	mediumFaceOptions := &opentype.FaceOptions{
		Size:    opt.MediumSize,
		DPI:     opt.DPI,
		Hinting: font.HintingFull,
	}
	mediumFace, err := opentype.NewFace(tt, mediumFaceOptions)
	if err != nil {
		return
	}
	smallFaceOptions := &opentype.FaceOptions{
		Size:    opt.SmallSize,
		DPI:     opt.DPI,
		Hinting: font.HintingFull,
	}
	smallFace, err := opentype.NewFace(tt, smallFaceOptions)
	if err != nil {
		return
	}

	largeFont = &Font{Face: largeFace, FaceOptions: largeFaceOptions}
	mediumFont = &Font{Face: mediumFace, FaceOptions: mediumFaceOptions}
	smallFont = &Font{Face: smallFace, FaceOptions: smallFaceOptions}

	return
}
