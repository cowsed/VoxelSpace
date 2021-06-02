package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	logMem = false
	logCpu = true
)

var (
	WindowWidth  int = 800
	WindowHeight int = 600
)

var redraw = true
var frameCount = 0

const (
	fontPath = "Font.ttf"
	fontSize = 14
)

var HeightMap *image.Gray
var ColorMap *image.RGBA

var SkyCol = sdl.Color{R: 255, G: 0, B: 255, A: 255}

func main() {
	//CPU Logging
	if logCpu {
		f, err := os.Create("cpu.pprof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	//Load Images
	heightFile, err := os.Open("Images/Height.png")
	if err != nil {
		panic(err)
	}
	var HeightMapTemp image.Image
	HeightMapTemp, err = png.Decode(heightFile)
	if err != nil {
		panic(err)
	}
	HeightMap = image.NewGray(HeightMapTemp.Bounds())
	draw.Draw(HeightMap, HeightMap.Bounds(), HeightMapTemp, image.Point{0, 0}, draw.Over)
	heightFile.Close()
	fmt.Printf("Loaded Height %dx%d\n", HeightMap.Rect.Dx(), HeightMap.Rect.Dy())

	//Load Color Map
	colorFile, err := os.Open("Images/Color.png")
	if err != nil {
		panic(err)
	}
	var ColorMapTemp image.Image
	ColorMapTemp, err = png.Decode(colorFile)
	if err != nil {
		panic(err)
	}
	ColorMap = image.NewRGBA(ColorMapTemp.Bounds())
	draw.Draw(ColorMap, ColorMap.Bounds(), ColorMapTemp, image.Point{0, 0}, draw.Over)

	colorFile.Close()
	fmt.Printf("Loaded Color %dx%d\n", ColorMap.Rect.Dx(), ColorMap.Rect.Dy())

	//Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	//Initialize Window
	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(WindowWidth), int32(WindowHeight), sdl.WINDOW_SHOWN+sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	//Initialize The Drawing Space
	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	//Initialize Font Lib
	if err = ttf.Init(); err != nil {
		return
	}
	defer ttf.Quit()

	// Load the font for our text
	var font *ttf.Font
	font, err = ttf.OpenFont(fontPath, fontSize)
	if err != nil {
		panic(err)
	}
	defer font.Close()

	// Create a text
	var text *sdl.Surface
	defer text.Free()

	window.UpdateSurface()

	running := true
	var lastFrameTime uint32 = 0
	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)
	speed := 1.5
	dbcol := sdl.Color{R: 255, G: 0, B: 255, A: 255}

	cam := Camera{
		Pos:      Point{0, 260},
		Angle:    0,
		Distance: 1000,
		Height:   100,
		Horizon:  50,
	}

	var keyMap = map[sdl.Keycode]bool{}
	var vel Point
	var renderWG sync.WaitGroup
	var NumThreads int = 4
	var renderControls []chan bool
	for i := range renderControls {
		renderControls[i] = make(chan bool)
	}
	for running {

		//Get Key Events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
			case *sdl.KeyboardEvent:
				keyCode := e.Keysym.Sym
				if e.State == sdl.PRESSED {
					keyMap[keyCode] = true
				} else if e.State == sdl.RELEASED {
					delete(keyMap, keyCode)
				}
			}

		}
		var modi float64 = 1
		//Handle Keys
		for k, down := range keyMap {
			redraw = true
			if !down {
				continue
			}
			vel = Point{0, 0}
			if k == sdl.K_LSHIFT {
				modi = 4
			}
			switch string(k) {
			case "w":
				vel.Y = -1
			case "s":
				vel.Y = 1
			case "q":
				vel.X = -1
			case "e":
				vel.X = 1
			case "z":
				cam.Height += 1 * modi
			case "x":
				cam.Height -= 9 * modi
			case "r":
				cam.Horizon += 4
			case "f":
				cam.Horizon -= 4

			case "a":
				cam.Angle -= 0.02
			case "d":
				cam.Angle += 0.02
			case "u":
				runtime.ReadMemStats(&gcStats)
			case "g":
				runtime.GC()

			}

			//Do Physics
			vel = vel.Rot(cam.Angle)
			vel = vel.Mul(speed * modi)
			cam.Pos = cam.Pos.Add(vel)

			//Min Height
			cam.Height = max(cam.Height, sampleHeight(cam.Pos.X, cam.Pos.Y)+10)

		}

		//Redraw things
		frameCount++
		if redraw {

			surface.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(WindowWidth), H: int32(WindowHeight)}, SkyCol.Uint32())

			pixels := surface.Pixels()
			renderWG.Add(NumThreads)
			DrawFrameChunk(0, WindowWidth, cam, WindowWidth, WindowHeight, pixels, surface)

			renderWG.Wait()
			redraw = false
		}

		millis := sdl.GetTicks() - lastFrameTime
		if millis > 16 {
			fmt.Println("LongFrame", millis)
			//runtime.ReadMemStats(&gcStats)
		}

		frameData := fmt.Sprintf("Frames: %d\n Delta %d\n %v \nMem: %vkb, NumGC: %d\n%v", frameCount, millis, cam, gcStats.Alloc/1000, gcStats.NumGC, keyMap)
		// Draw the text
		DrawTextBoxToSurface(frameData, 0, 0, 300, dbcol, font, text, surface)
		window.UpdateSurface()
		surface.Free()

		lastFrameTime = sdl.GetTicks()

	}

	//Memory Profiling
	if logMem {
		f, err := os.Create("mem.pprof")
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}

func sampleHeight(x, y float64) float64 {
	//pts := []Point{{x + 1, y}, {x - 1, y}, {x, y + 1}, {x, y - 1}, {x, y}}
	//var val float64 = 0
	//for _, p := range pts {
	//	val += float64(HeightMap.GrayAt((int(p.X)), (int(p.Y))).Y)
	//}
	//return val / 5
	h := HeightMap.GrayAt(int(x), int(y))
	return float64(h.Y)
}
func sampleColor(x, y float64) sdl.Color {
	c := ColorMap.RGBAAt(FAbs(int(x), 120), FAbs(int(y), 120))
	return sdl.Color{R: c.R, G: c.G, B: c.B, A: c.A}
}
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

//https://github.com/s-macke/VoxelSpace
func DrawFrameChunk(startX, endX int, c Camera, screen_width, screen_height int, pixels []byte, surface *sdl.Surface) {
	//For some reason this works when negative but when positive the camera controller goes the wrong way
	var sinang = math.Sin(-c.Angle)
	var cosang = math.Cos(-c.Angle)

	var hiddeny = make([]int, screen_width)
	for i := 0; i < int(screen_width); i++ {
		hiddeny[i] = screen_height
	}

	var deltaz = 1.0
	var plx, ply, prx, pry, dx, dy, invz float64
	var sampleH, heightonscreen float64
	var samplePoint Point
	//var screen_height, screen_width = float64()
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

		for i := startX; i < endX; i++ {
			samplePoint = Point{math.Floor(plx), math.Floor(ply)}
			sampleH = sampleHeight(samplePoint.X, samplePoint.Y)
			heightonscreen = (c.Height-sampleH)*invz + c.Horizon
			DrawVerticalLine(i, int(heightonscreen), hiddeny[i], sampleColor(samplePoint.X, samplePoint.Y), pixels, surface)
			if int(heightonscreen) < hiddeny[i] {
				hiddeny[i] = int(heightonscreen)
			}
			plx += dx
			ply += dy
		}
		deltaz += 0.005
	}
}

//Draws specified data to text Surface then to surface Surface
//Draws at (x,y) with a text box that is w wide
func DrawTextBoxToSurface(data string, x, y int32, w int, col sdl.Color, font *ttf.Font, text, surface *sdl.Surface) {
	var err error
	text.Free()
	if text, err = font.RenderUTF8BlendedWrapped(data, col, w); err != nil {
		fmt.Println(err, "Err")

	}
	surface.FillRect(&sdl.Rect{X: x, Y: y, W: text.W, H: text.H}, 0x00404040)

	if err = text.Blit(nil, surface, &sdl.Rect{X: x, Y: y, W: 0, H: 0}); err != nil {
		fmt.Println(err, "Err")
	}
	text.Free()

}

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
