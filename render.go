package main

import (
	"math"
	"sync"

	"github.com/veandco/go-sdl2/sdl"
)

func sampleHeight(x, y float64) float64 {
	h := HeightMap.GrayAt(int(x), int(y))
	return float64(h.Y) * float64(terrainScale)
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

//https://github.com/s-macke/VoxelSpace
func DrawFrameChunk(startX, endX int, hiddeny []int, c Camera, screen_width, screen_height int, pixels []byte, surface *sdl.Surface) {
	//For some reason this works when negative but when positive the camera controller goes the wrong way
	var sinang = math.Sin(-c.Angle)
	var cosang = math.Cos(-c.Angle)

	for i := startX; i <= endX; i++ {
		hiddeny[i-startX] = screen_height
	}

	var deltaz = 1.0
	var plx, ply, prx, pry, dx, dy, invz float64
	var sampleH, heightonscreen float64
	var samplePoint Point
	// Draw from front to back
	for z := 1.0; z < c.Distance; z += deltaz {
		// 90 degree field of view
		plx = -cosang*z - sinang*z
		ply = sinang*z - cosang*z
		prx = cosang*z - sinang*z
		pry = -sinang*z - cosang*z
		dx = (prx - plx) / float64(screen_width)
		dy = (pry - ply) / float64(screen_width)
		plx += c.Pos.X
		ply += c.Pos.Y
		invz = 1. / z * 240.
		//Set up for multi thread rendering
		plx += dx * float64(startX)
		ply += dy * float64(startX)
		for i := startX; i <= endX; i++ {
			samplePoint = Point{math.Floor(plx), math.Floor(ply)}
			sampleH = sampleHeight(samplePoint.X, samplePoint.Y)
			heightonscreen = (c.Height-sampleH)*invz + c.Horizon
			DrawVerticalLine(i, int(heightonscreen), hiddeny[i-startX], sampleColor(samplePoint.X, samplePoint.Y), pixels, surface)
			if int(heightonscreen) < hiddeny[i-startX] {
				hiddeny[i-startX] = int(heightonscreen)
			}
			plx += dx
			ply += dy
		}
		deltaz += 0.005
	}
	//Fill the rest of the screen with skycol
	for i := 0; i < len(hiddeny); i++ {
		start := hiddeny[i]
		DrawVerticalLine(i+startX, 0, start, SkyCol, pixels, surface)
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
