package main

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

//Logging Information
const (
	logMem = false
	logCpu = false
)

//Window Information
var (
	WindowWidth  int = 800
	WindowHeight int = 600
)

//Rendering Information
var (
	redraw      = true //Something changed and should redraw
	frameCount  = 0    //Number of frames rendered
	totalMillis int    //Number of milliseconds used by rendering
)

//Physics Information
var (
	speed                 = 4.0
	SpeedModifier float64 = 1 //Speed Modifier
	playerHeight          = 15
	doGravity             = false
	GForce                = 0.04
)

//Font Information
const (
	fontPath = "Font.ttf"
	fontSize = 14
)

var fontColor = sdl.Color{R: 255, G: 0, B: 255, A: 255}

//World Data
var (
	HeightMap *image.Gray
	ColorMap  *image.RGBA
	SkyCol    = sdl.Color{R: 155, G: 255, B: 255, A: 255}
)

//Multi Thread Rendering Information
var (
	renderWG       sync.WaitGroup                                //Wait for render threads
	NumThreads     int                = 10                       //Number of threads to split the rendering between
	draw_unit                         = WindowWidth / NumThreads //How many lines does each goroutine render
	renderControls []chan RenderStuff                            //Sending information to rendering threads
)

func main() {
	//Initialize CPU Profiling
	StartCPUProfile()
	//Load Images
	LoadAll()

	//Initialize SDL
	err := sdl.Init(sdl.INIT_EVERYTHING)
	check(err)
	defer sdl.Quit()

	//Initialize Window
	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(WindowWidth), int32(WindowHeight), sdl.WINDOW_SHOWN)
	check(err)
	defer window.Destroy()

	//Initialize The Drawing Space
	surface, err := window.GetSurface()
	check(err)

	//Initialize Font Lib
	err = ttf.Init()
	check(err)
	defer ttf.Quit()

	// Load the font for our text
	var font *ttf.Font
	font, err = ttf.OpenFont(fontPath, fontSize)
	check(err)
	defer font.Close()

	// Create a text
	var text *sdl.Surface
	defer text.Free()

	defer UISurf.Free()

	RenderUI(font)
	window.UpdateSurface()

	running := true
	var lastFrameTime uint32 = 0
	var gcStats runtime.MemStats

	cam := Camera{
		Pos:      Point{0, 260},
		Angle:    0,
		Distance: 2000,
		Height:   100,
		Horizon:  50,
	}
	var keyMap = map[sdl.Keycode]bool{} //Map of keys and if theyre pressed
	var InputVel Point                  //Velocity of camera
	//var Vel Point
	var VelZ float64
	renderControls = make([]chan RenderStuff, NumThreads) //sends reference to pixels to fill
	for i := range renderControls {
		renderControls[i] = make(chan RenderStuff)
		go func(i int, currentChan chan RenderStuff) {
			HandleRenderThread(i*draw_unit, (i+1)*draw_unit-1, currentChan, &cam)
		}(i, renderControls[i])
	}
	fmt.Println("Beginning Rendering")
	for running {
		start := time.Now()
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
					switch keyCode {
					case sdl.K_UP:
						if UISelected > 0 {
							UISelected--
						}
					case sdl.K_DOWN:
						if UISelected < len(UIItems)-1 {
							UISelected++
						}
					case sdl.K_LEFT:
						UIItems[UISelected].PreviousItem()
					case sdl.K_RIGHT:
						UIItems[UISelected].NextItem()
					}
				} else if e.State == sdl.RELEASED {
					delete(keyMap, keyCode)
				}
			}

		}
		//Handle Keys
		InputVel = Point{0, 0}

		for k, down := range keyMap {
			redraw = true
			if !down {
				continue
			}

			switch string(k) {
			case "w":
				InputVel.Y = -1
			case "s":
				InputVel.Y = 1
			case "q":
				InputVel.X = -1
			case "e":
				InputVel.X = 1
			case "z":
				cam.Height += 3
			case "x":
				cam.Height -= 3
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

		}

		//Do Physics
		InputVel = InputVel.Rot(cam.Angle)
		InputVel = InputVel.Mul(speed * SpeedModifier)
		cam.Pos = cam.Pos.Add(InputVel)

		//Min Height
		h := sampleHeight(cam.Pos.X, cam.Pos.Y) + float64(playerHeight)
		cam.Height = max(cam.Height, h)
		if doGravity {
			if cam.Height > h {
				VelZ -= GForce
			} else {
				VelZ = 0
			}
			cam.Height += VelZ
			redraw = true
		}
		//Redraw things
		var millis uint32
		if redraw {
			frameCount++
			//Clear Screen
			//surface.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(WindowWidth), H: int32(WindowHeight)}, 0xffffff)

			//Send Rendering Information
			pixels := surface.Pixels()
			renderWG.Add(NumThreads)
			for i := range renderControls {
				renderControls[i] <- RenderStuff{
					pix:           pixels,
					surface:       surface,
					screen_width:  WindowWidth,
					screen_height: WindowHeight,
					wg:            &renderWG,
				}
			}
			//Wait for all the rendering threades to finish
			renderWG.Wait()
			millis = sdl.GetTicks() - lastFrameTime
			totalMillis += int(millis)

			RenderUI(font)
			redraw = false

		}

		// Draw DB text
		frameData := fmt.Sprintf("Frames: %d\n Delta %d\nAvgDelta: %d\n Modifier: %.3f\n%v \nMem: %vkb, NumGC: %d\n%v", frameCount, millis, totalMillis/frameCount, SpeedModifier, cam, gcStats.Alloc/1000, gcStats.NumGC, keyMap)
		DrawTextBoxToSurface(frameData, 0, 0, 300, fontColor, font, text, surface)

		ShowUI(surface)
		//Update frame
		window.UpdateSurface()
		surface.Free()

		lastFrameTime = sdl.GetTicks()

		past := time.Since(start)

		time.Sleep(16*time.Millisecond - past)
	}

	//Memory Profiling
	ProfileMemory()
	//Finish CPU Profiling
	pprof.StopCPUProfile()
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
