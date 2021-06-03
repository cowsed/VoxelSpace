package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

func StartCPUProfile() {
	//CPU Logging
	if logCpu {
		f, err := os.Create("cpu.pprof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}
}
func ProfileMemory() {
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

func LoadAll() {
	//Load Images
	heightFile, err := os.Open("Images/World.png")
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
	colorFile, err := os.Open("Images/World.png")
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
}
