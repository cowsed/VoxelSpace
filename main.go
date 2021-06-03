package main

import (
	"fmt"
	"image"
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
	WindowWidth  int = 960
	WindowHeight int = 640
	captureMouse     = false
	keyMap           = map[sdl.Keycode]bool{} //Map of keys and if theyre pressed

)

//Rendering Information
var (
	redraw      = true //Something changed and should redraw
	frameCount  = 0    //Number of frames rendered
	totalMillis int    //Number of milliseconds used by rendering
	drawDB      = true
)

//Physics Controls
var (
	speed         float64 = 4.0
	SpeedModifier float64 = 1 //Speed Modifier
	playerHeight          = 15
	doGravity             = false
	GForce                = 0.06
)

//Physics Variables
var (
	VelZ     float64
	InputVel Point //Velocity of cameras
)

//Font Information
const (
	fontPath = "Font.ttf"
	fontSize = 14
)

var fontColor = sdl.Color{R: 255, G: 0, B: 255, A: 255}

//World Data
var (
	terrainScale float64 = 1
	HeightMap    *image.Gray
	ColorMap     *image.RGBA
	SkyCol               = sdl.Color{R: 110, G: 210, B: 255, A: 255}
	FogAmt       float64 = .3
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
	sdl.CaptureMouse(captureMouse)
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
						UISelected--
						if UISelected < 0 {
							UISelected += len(UIItems)
						}
					case sdl.K_DOWN:
						UISelected++
						UISelected %= len(UIItems)
					case sdl.K_LEFT:
						UIItems[UISelected].PreviousItem()
					case sdl.K_RIGHT:
						UIItems[UISelected].NextItem()
					case sdl.K_ESCAPE:
						captureMouse = !captureMouse
						sdl.CaptureMouse(captureMouse)
						//window.SetGrab(captureMouse)
						sdl.SetRelativeMouseMode(captureMouse)
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
			case "a":
				InputVel.X = -1
			case "d":
				InputVel.X = 1
			case "z":
				cam.Height += 3
			case "x":
				cam.Height -= 3
			case "r":
				cam.Horizon += 4
			case "f":
				cam.Horizon -= 4

			case "q":
				cam.Angle -= 0.02
			case "e":
				cam.Angle += 0.02
			case "u":
				runtime.ReadMemStats(&gcStats)
			case "g":
				runtime.GC()
			}

		}
		//Handle Mouse

		deltaX, deltaY, _ := sdl.GetRelativeMouseState()

		cam.Angle += float64(deltaX) * .005
		cam.Horizon -= float64(deltaY) * 2
		if deltaX > 0 || deltaY > 0 {
			redraw = true
		}
		//sdl.WarpMouseGlobal(newMouseX%int32(WindowWidth), newMouseY%int32(WindowHeight))

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
				cam.Height += VelZ
				redraw = true
			} else {
				VelZ = 0
			}

		}
		//Redraw things
		var millis uint32
		if redraw {
			frameCount++
			//Clear Screen

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
			//Wait for all the rendering threads to finish
			renderWG.Wait()
			millis = sdl.GetTicks() - lastFrameTime
			totalMillis += int(millis)

			RenderUI(font)
			redraw = false

		}

		// Draw DB text
		past := time.Since(start)

		if drawDB {
			frameData := fmt.Sprintf("Frames: %d\n Delta %d\nAvgDelta: %d\nRenderTime: %dms\n%v \nMem: %vkb, NumGC: %d\n%v", frameCount, millis, totalMillis/frameCount, past.Milliseconds(), cam, gcStats.Alloc/1000, gcStats.NumGC, keyMap)
			DrawTextBoxToSurface(frameData, 0, 0, 300, fontColor, font, text, surface)
		}
		ShowUI(surface)
		//Update frame
		window.UpdateSurface()
		surface.Free()

		lastFrameTime = sdl.GetTicks()

		past = time.Since(start)

		time.Sleep(16*time.Millisecond - past)
	}

	//Memory Profiling
	ProfileMemory()
	//Finish CPU Profiling
	pprof.StopCPUProfile()
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
