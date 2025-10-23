package server

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

const (
	screenWidth  = 800
	screenHeight = 480
	runesPerLine = 42
	maxLines     = 13
)

//go:embed "font/White Rabbit.ttf"
var fontData []byte

var (
	fontDescription = FontDescription{
		Font:    mustReadFont(),
		DPI:     72.0,
		Hinting: font.HintingNone,
		Size:    32.0,
		Ratio:   0.65,
		Offset:  10,
	}
	PaletteBlackWhite = color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.White,
	}
	StandardBounds = image.Rect(0, 0, screenWidth, screenHeight)
)

type FontDescription struct {
	Font    *truetype.Font
	DPI     float64
	Hinting font.Hinting
	Size    float64
	Ratio   float64
	Offset  int
}

// PrepareFreetypeContext sets up all the bits and bobs related to drawing text on the image
func PrepareFreetypeContext(dst *image.Paletted, src image.Image, font FontDescription) (*freetype.Context, int) {
	c := freetype.NewContext()
	c.SetDPI(font.DPI)
	c.SetFont(font.Font)
	c.SetHinting(font.Hinting)
	c.SetFontSize(font.Size)
	c.SetSrc(src)
	c.SetDst(dst)
	c.SetClip(dst.Bounds())

	baseline := (int(c.PointToFixed(font.Size) >> 6))

	return c, baseline
}

func mustReadFont() *truetype.Font {
	f, err := truetype.Parse(fontData)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	return f
}

// mustDrawText draws the given text in the given context, at the given location
func mustDrawText(c *freetype.Context, x int, y int, text string) {
	pt := freetype.Pt(x, y)
	_, err := c.DrawString(text, pt)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func CreateScreen(text []string) *image.Paletted {

	img := image.NewPaletted(StandardBounds, PaletteBlackWhite)

	ctx, baseline := PrepareFreetypeContext(img, &image.Uniform{PaletteBlackWhite[1]}, fontDescription)
	for i, line := range text {
		if i > maxLines {
			break
		}
		runes := []rune(line) // Sometimes runes are more than one byte, and len(string) returns number of *bytes*
		if len(runes) > runesPerLine {
			line = string(runes[:runesPerLine])
		}
		mustDrawText(ctx, 15, (i*baseline)+baseline+fontDescription.Offset, line)
	}

	return img

}
