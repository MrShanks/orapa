package game

import (
	"image/color"
	"math"
)

type Shape struct {
	basePoints    []GridPoint
	localPoints   []GridPoint
	gridX, gridY  int
	clr           color.Color
	logicalColor  string
	rotationSteps int
	board         int
}

func NewShape(points []GridPoint, x, y int, clr color.Color, logColor string, board int) *Shape {
	s := &Shape{
		basePoints:   points,
		gridX:        x,
		gridY:        y,
		clr:          clr,
		logicalColor: logColor,
		board:        board,
	}
	s.applyRotation()
	return s
}

func (s *Shape) applyRotation() {
	s.localPoints = make([]GridPoint, len(s.basePoints))
	for i, bp := range s.basePoints {
		p := bp
		for range s.rotationSteps {
			p = GridPoint{X: -p.Y, Y: p.X}
		}
		s.localPoints[i] = p
	}

	minX, minY := s.localPoints[0].X, s.localPoints[0].Y
	for _, p := range s.localPoints {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
	}

	for i := range s.localPoints {
		s.localPoints[i].X -= minX
		s.localPoints[i].Y -= minY
	}
}

func (s *Shape) Rotate() {
	s.rotationSteps = (s.rotationSteps + 1) % 4
	s.applyRotation()
}

func (s *Shape) Bounds() (int, int, int, int) {
	if len(s.localPoints) == 0 {
		return s.gridX, s.gridY, s.gridX, s.gridY
	}
	minX, maxX := s.localPoints[0].X, s.localPoints[0].X
	minY, maxY := s.localPoints[0].Y, s.localPoints[0].Y
	for _, p := range s.localPoints {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return s.gridX + int(math.Round(minX)), s.gridY + int(math.Round(minY)), s.gridX + int(math.Round(maxX)), s.gridY + int(math.Round(maxY))
}

func (s *Shape) IsValidPosition() bool {
	minX, minY, maxX, maxY := s.Bounds()
	if minX < 0 || maxX > cols {
		return false
	}
	if s.board == 1 {
		return minY >= 0 && maxY <= rows
	}
	return minY >= 0 && maxY <= rows+8
}
