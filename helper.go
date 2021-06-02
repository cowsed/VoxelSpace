package main

import (
	"fmt"
	"math"
)

type Point struct {
	X, Y float64
}

func (p Point) Add(b Point) Point {
	return Point{
		X: p.X + b.X,
		Y: p.Y + b.Y,
	}

}
func (p Point) Mul(s float64) Point {
	return Point{
		X: p.X * s,
		Y: p.Y * s,
	}
}
func (p Point) Rot(a float64) Point {
	return Point{
		X: p.X*math.Cos(a) - p.Y*math.Sin(a),
		Y: p.X*math.Sin(a) + p.Y*math.Cos(a),
	}
}

type Camera struct {
	Pos      Point
	Angle    float64
	Distance float64
	Height   float64
	Horizon  float64
}

func (c Camera) String() string {
	return fmt.Sprintf("P: (%.3f, %.3f) \nθ: %.3fpi \nDist: %f \nHeight: %f \nHorizon: %f", c.Pos.X, c.Pos.Y, c.Angle/math.Pi, c.Distance, c.Height, c.Horizon)
}

//Math Things
//=============

//Abs value and Fract
func FAbs(n, mod int) int {
	var v int = n
	//if n < 0 {
	//		v = -n
	//}
	return v //- (v / mod)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

//Classic Error Fail for loading
func check(err error) {
	if err != nil {
		panic(err)
	}
}
