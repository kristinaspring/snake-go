package main

import (
	"container/list"
	"fmt"
	"image/color"
	"math"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
)

type Direction int

const (
	None Direction = iota
	Up
	Down
	Right
	Left
)

const (
	DefaultSquareSize      = 10
	DefaultBuffer          = 10
	DefaultPixelsPerSecond = 10
	DefaultStartingFrames  = 12
	DefaultFramesToGrow    = 4
	DefaultThreshold       = 5.0
)

type point interface {
	X() float64
	Y() float64
}

type location struct {
	x float64
	y float64
}

func (l location) X() float64 {
	return l.x
}

func (l location) Y() float64 {
	return l.y
}

func (l location) Equal(other location) bool {
	return int(l.x) == int(other.x) && int(l.y) == int(other.y)
}

type Edges struct {
	left   float64
	right  float64
	top    float64
	bottom float64
}

type Snake struct {
	config SnakeConfig

	currDirection         Direction
	currDirectionStartLoc location
	nextDirection         Direction
	locations             *list.List
	currDrawing           *imdraw.IMDraw
	grow                  int
	score                 int

	item       tracker
	otherSnake tracker
}

type SnakeConfig struct {
	Edges            Edges
	StartingPosition point
	SquareSize       float64
	TaperTo          float64
	Buffer           float64
	Colors           []color.Color
	PixelsPerSec     float64
	StartingFrames   int
	FramesToGrow     int
	Threshold        float64
}

func NewSnake(itemTracker tracker, config SnakeConfig) *Snake {
	c := validateConfig(config)

	l := list.New()

	item := itemTracker
	if item == nil {
		item = defaultTracker{}
	}

	s := &Snake{
		config:    c,
		locations: l,
		item:      item,
	}
	s.Reset(nil)
	return s
}

func validateConfig(config SnakeConfig) SnakeConfig {
	c := config
	e := c.Edges
	if c.Edges.right < c.Edges.left {
		e.right = c.Edges.left
		e.left = c.Edges.right
	}
	if c.Edges.top < c.Edges.bottom {
		e.top = c.Edges.bottom
		e.bottom = c.Edges.top
	}
	c.Edges = e

	if c.StartingPosition == nil || c.StartingPosition.X() < 0 || c.StartingPosition.Y() < 0 {
		middleY := (e.top-e.bottom)/2.0 + e.bottom
		middleX := (e.right-e.left)/2.0 + e.left
		c.StartingPosition = location{x: middleX, y: middleY}
	}

	if c.SquareSize <= 0 {
		c.SquareSize = DefaultSquareSize
	}

	if c.Buffer <= 0 {
		c.Buffer = DefaultBuffer
	}

	if c.Colors == nil || len(c.Colors) == 0 {
		c.Colors = []color.Color{color.RGBA{0x00, 0x00, 0x00, 0xff}}
	}

	if c.PixelsPerSec <= 0 {
		c.PixelsPerSec = DefaultPixelsPerSecond
	}

	if c.StartingFrames <= 0 {
		c.StartingFrames = DefaultStartingFrames
	}

	if c.FramesToGrow < 0 {
		c.FramesToGrow = DefaultFramesToGrow
	}

	if c.Threshold <= 0 {
		c.Threshold = DefaultThreshold
	}

	return c
}

func (s *Snake) SetOtherSnake(other tracker) {
	if other != nil {
		s.otherSnake = other
	}
}

func (s *Snake) SetDirection(d Direction) {
	// don't let the snake do a 180 turn
	if s.currDirection == Up && d == Down ||
		s.currDirection == Down && d == Up ||
		s.currDirection == Left && d == Right ||
		s.currDirection == Right && d == Left {
		fmt.Println("Can't do")
		return
	}
	s.nextDirection = d
}

func (s *Snake) At(l location) bool {
	return pointInList(l, s.locations)
}

func (s *Snake) Paint() *imdraw.IMDraw {
	newDrawing := imdraw.New(nil)
	newDrawing.EndShape = imdraw.SharpEndShape

	ss := s.config.SquareSize
	b := s.config.Buffer

	sLen := float64(s.locations.Len())

	e := s.locations.Back()
	i := int(math.Mod(math.Round(sLen/2), float64(len(s.config.Colors))))
	r := s.config.TaperTo / 2.0
	rDelta := (s.config.SquareSize - s.config.TaperTo) / sLen
	for e != nil {
		l := e.Value.(point)

		if i < 0 {
			i = len(s.config.Colors) - 1
		}
		newDrawing.Color = s.config.Colors[i]
		// newDrawing.Push(pixel.Vec{X: s.buffer + l.X()*s.squareSize, Y: s.buffer + l.Y()*s.squareSize}, pixel.Vec{X: s.buffer + (l.X() * s.squareSize) + s.squareSize, Y: s.buffer + (l.Y() * s.squareSize) + s.squareSize})
		newDrawing.Push(pixel.Vec{X: b + l.X()*ss + ss/2, Y: b + l.Y()*ss + ss/2})
		newDrawing.Circle(r, 0)
		e = e.Prev()
		if e != nil {
			e = e.Prev()
		} else {
			r -= rDelta / 2
		}
		r += rDelta
		i--
	}
	return newDrawing
}

func (s *Snake) Tick(t float64, deltaT float64) {
	h := s.locations.Front().Value.(point)
	newX := h.X()
	newY := h.Y()

	pps := s.config.PixelsPerSec

	switch s.currDirection {
	case Up:
		newY = h.Y() + pps*deltaT
	case Down:
		newY = h.Y() - pps*deltaT
	case Left:
		newX = h.X() - pps*deltaT
	case Right:
		newX = h.X() + pps*deltaT
	}

	threshold := s.config.Threshold
	if s.nextDirection != None && s.nextDirection != s.currDirection {
		ss := s.config.SquareSize
		xCheck := math.Mod(newX, ss)
		yCheck := math.Mod(newY, ss)

		xRound := math.Round(newX)
		yRound := math.Round(newY)
		// switch directions if we're ready
		if (xCheck < threshold || (ss-xCheck) < threshold || yCheck < threshold || (ss-yCheck) < threshold) &&
			(math.Abs(xRound-s.currDirectionStartLoc.X()) >= 1 || math.Abs(yRound-s.currDirectionStartLoc.Y()) >= 1) {
			s.currDirection = s.nextDirection
			s.nextDirection = None
			newX = xRound
			newY = yRound
			s.currDirectionStartLoc = location{x: newX, y: newY}
		}
	}

	// check that the new spot won't be outside of the game board
	edges := s.config.Edges
	if int(newY) < int(edges.bottom) || int(newY) >= int(edges.top) || int(newX) < int(edges.left) || int(newX) >= int(edges.right) {
		s.Reset(nil)
		return
	}

	// if we're currently going nowhere, we're done here
	if s.currDirection == None {
		return
	}

	newSquare := location{x: newX, y: newY}

	// check for collisions with the other snake
	if s.otherSnake != nil && s.otherSnake.At(newSquare) {
		s.Reset(nil)
		fmt.Println("collided with other snake. dead")
		return
	}

	// check for collisions with itself
	e := s.locations.Front()
	// skip the first few, those will be too close
	for i := 0; i < 5; i++ {
		if e != nil {
			e = e.Next()
		}
	}
	for e != nil {
		l := e.Value.(point)
		if math.Abs(l.X()-newX) < 0.3 && math.Abs(l.Y()-newY) < 0.3 {
			s.Reset(nil)
			fmt.Println("killed self")
			return
		}
		e = e.Next()
	}

	// add new item to the list
	s.locations.PushFront(newSquare)

	// check if we ate something and if so, don't remove the last item from the
	// list.
	if s.item.At(newSquare) {
		s.item.Reset(s.locations)
		s.grow += s.config.FramesToGrow
		s.score++
	}

	if s.grow > 0 {
		s.grow--
		return
	}

	// remove the last item from the list
	s.locations.Remove(s.locations.Back())
}

// game is lost, bring everything back to the beginning
func (s *Snake) Reset(_ *list.List) {
	s.currDirection = None
	s.nextDirection = None
	s.locations.Init()
	s.locations.PushFront(s.config.StartingPosition)
	s.currDirectionStartLoc = location{x: s.config.StartingPosition.X() - 2.0, y: s.config.StartingPosition.Y() - 2.0}
	s.grow = s.config.StartingFrames
	s.score = 0
}
