package main

import (
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

func genHeight(x, y float64) float64 {
	return math.Sin(x/10) * 10
}
func genCol(x, y float64) sdl.Color {
	return sdl.Color{
		R: uint8((1.0 + math.Sin(x/10.0)) / 2.0 * 255.0),
		A: 255,
	}
}
