package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var UISurf *sdl.Surface
var UIPosX int = 400
var UIPosY int
var UISelected int

var UIItems []UIItem = []UIItem{
	&BoolEdit{"ShowDB", &drawDB},
	&BoolEdit{"DoGravity", &doGravity},
	&FloatEdit{"GForce", &GForce},
	&FloatEdit{"Speed", &SpeedModifier},
	&IntEdit{"Player Height", &playerHeight},
	&FloatEdit{"Terrain Scale", &terrainScale},
	&FloatEdit{"Fog Amount", &FogAmt},

	&IntEdit{"UI.x", &UIPosX},
	&IntEdit{"UI.y", &UIPosY},
}

func RenderUI(font *ttf.Font) {
	UISurf.Free()
	col := sdl.Color{R: 255, G: 0, B: 0, A: 255}
	text := fmt.Sprintf("UI - %d\n", UISelected)
	for i := range UIItems {
		if i == UISelected {
			text += "> "
		}
		text += UIItems[i].String() + "\n"
	}
	var err error
	UISurf, err = font.RenderUTF8BlendedWrapped(text, col, 300)

	check(err)
}

func ShowUI(surface *sdl.Surface) {
	uiRect := UISurf.ClipRect
	uiRect.X += int32(UIPosX)
	uiRect.Y += int32(UIPosY)
	//Draw UI
	surface.FillRect(&uiRect, 0xff000000)

	UISurf.Blit(nil, surface, &uiRect)

}

type UIItem interface {
	String() string
	NextItem()
	PreviousItem()
}

type BoolEdit struct {
	Name  string
	Value *bool
}

func (b *BoolEdit) String() string {
	return fmt.Sprintf("%s: \t<  %v  >", b.Name, *b.Value)
}

func (b *BoolEdit) NextItem() {
	*b.Value = !*b.Value
}
func (b *BoolEdit) PreviousItem() {
	*b.Value = !*b.Value
}

type FloatEdit struct {
	Name  string
	Value *float64
}

func (f *FloatEdit) String() string {
	return fmt.Sprintf("%s: \t<  %f  >", f.Name, *f.Value)
}
func (f *FloatEdit) NextItem() {
	*f.Value += 0.01
}
func (f *FloatEdit) PreviousItem() {
	*f.Value -= 0.01
}

type IntEdit struct {
	Name  string
	Value *int
}

func (i *IntEdit) String() string {
	return fmt.Sprintf("%s: \t<  %d  >", i.Name, *i.Value)
}
func (i *IntEdit) NextItem() {
	*i.Value += 1
}
func (i *IntEdit) PreviousItem() {
	*i.Value -= 1
}
