package main

import (
	"sync"

	"github.com/veandco/go-sdl2/sdl"
)

func sampleHeight(x, y float64) float64 {
	h := HeightMap.GrayAt(int(x), int(y))
	return float64(h.Y)
}
func sampleColor(x, y float64) sdl.Color {
	c := ColorMap.RGBAAt(FAbs(int(x), 120), FAbs(int(y), 120))
	return sdl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}

//Information given to a render thread when it needs to render
type RenderStuff struct {
	pix           []byte
	surface       *sdl.Surface
	screen_width  int
	screen_height int
	wg            *sync.WaitGroup
}

//HandleRenderThread keeps track of a render thread and renders with DrawFrameChunk when necessary
func HandleRenderThread(startX, endX int, comm chan RenderStuff, c *Camera) {
	var currentRender RenderStuff
	var pix []byte
	var surf *sdl.Surface
	var screen_height, screen_width int
	var hiddeny = make([]int, endX-startX+1)
	//Begin the rendering
	for {
		currentRender = <-comm //Wait for render instruction
		//
		pix = currentRender.pix
		surf = currentRender.surface
		screen_height = currentRender.screen_height
		screen_width = currentRender.screen_width
		DrawFrameChunk(startX, endX, hiddeny, *c, screen_width, screen_height, pix, surf)
		currentRender.wg.Done()
	}
}

//DrawVerticalLine at x from y to height with color
//Drawn into pixels
func DrawVerticalLine(x, y, height int, color sdl.Color, pixels []byte, surface *sdl.Surface) {
	if y > height {
		return
	}
	if y < 0 {
		y = 0
	}
	var z int
	for z = y; z <= height; z++ {
		setPixel(int32(x), int32(z), color, pixels, surface)
	}
}

//Sets a pixel of an surface (referred to by the slice )
func setPixel(x, y int32, col sdl.Color, pixels []byte, surface *sdl.Surface) {
	if x >= surface.W || y >= surface.H {
		return
	}
	if x < 0 || y < 0 {
		return
	}

	pos := y*surface.Pitch + x*int32(surface.BytesPerPixel())
	pixels[pos] = col.B
	pixels[pos+1] = col.G
	pixels[pos+2] = col.R
	pixels[pos+3] = col.A

}
