package game

import (
	"image/color"
	"math"
)

type GridPoint struct {
	X, Y float64
}

type Shape struct {
	basePoints    []GridPoint
	localPoints   []GridPoint
	gridX, gridY  int
	clr           color.Color
	logicalColor  string
	rotationSteps int
	flipped       bool
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
		if s.flipped {
			p.X = -p.X
		}
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

func (s *Shape) Flip() {
	s.flipped = !s.flipped
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

func (s *Shape) GlobalPoints() []GridPoint {
	pts := make([]GridPoint, len(s.localPoints))
	for i, p := range s.localPoints {
		pts[i] = GridPoint{X: float64(s.gridX) + p.X, Y: float64(s.gridY) + p.Y}
	}
	return pts
}

func onSegmentPoint(p, a, b GridPoint) bool {
	cross := (p.Y-a.Y)*(b.X-a.X) - (p.X-a.X)*(b.Y-a.Y)
	if math.Abs(cross) > 1e-4 {
		return false
	}
	dot := (p.X-a.X)*(b.X-a.X) + (p.Y-a.Y)*(b.Y-a.Y)
	if dot < -1e-4 {
		return false
	}
	sqLen := (b.X-a.X)*(b.X-a.X) + (b.Y-a.Y)*(b.Y-a.Y)
	if dot > sqLen+1e-4 {
		return false
	}
	return true
}

func isPointStrictlyInsidePolygon(p GridPoint, poly []GridPoint) bool {
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		if onSegmentPoint(p, a, b) {
			return false
		}
	}
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if ((poly[i].Y > p.Y) != (poly[j].Y > p.Y)) &&
			(p.X < (poly[j].X-poly[i].X)*(p.Y-poly[i].Y)/(poly[j].Y-poly[i].Y)+poly[i].X) {
			inside = !inside
		}
	}
	return inside
}

func orientation(p, q, r GridPoint) int {
	val := (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
	if math.Abs(val) < 0.0001 {
		return 0
	}
	if val > 0 {
		return 1
	}
	return 2
}

func segmentsCrossStrictly(a, b, c, d GridPoint) bool {
	o1 := orientation(a, b, c)
	o2 := orientation(a, b, d)
	o3 := orientation(c, d, a)
	o4 := orientation(c, d, b)
	return o1 != o2 && o3 != o4 && o1 != 0 && o2 != 0 && o3 != 0 && o4 != 0
}

func segmentsCollinearOverlap(a, b, c, d GridPoint) bool {
	o1 := orientation(a, b, c)
	o2 := orientation(a, b, d)
	o3 := orientation(c, d, a)
	o4 := orientation(c, d, b)
	if o1 == 0 && o2 == 0 && o3 == 0 && o4 == 0 {
		dirX, dirY := b.X-a.X, b.Y-a.Y
		lenAB := math.Hypot(dirX, dirY)
		if lenAB < 1e-4 {
			return false
		}
		dirX /= lenAB
		dirY /= lenAB

		projA, projB := 0.0, lenAB
		projC := (c.X-a.X)*dirX + (c.Y-a.Y)*dirY
		projD := (d.X-a.X)*dirX + (d.Y-a.Y)*dirY

		minCD := math.Min(projC, projD)
		maxCD := math.Max(projC, projD)

		overlapLen := math.Min(projB, maxCD) - math.Max(projA, minCD)
		return overlapLen > 1e-4
	}
	return false
}

func shapesOverlapOrEdgeTouch(s1, s2 *Shape) bool {
	p1 := s1.GlobalPoints()
	p2 := s2.GlobalPoints()

	for _, pt := range p1 {
		if isPointStrictlyInsidePolygon(pt, p2) {
			return true
		}
	}
	for _, pt := range p2 {
		if isPointStrictlyInsidePolygon(pt, p1) {
			return true
		}
	}

	for i := range p1 {
		a, b := p1[i], p1[(i+1)%len(p1)]
		for j := range p2 {
			c, d := p2[j], p2[(j+1)%len(p2)]
			if segmentsCrossStrictly(a, b, c, d) {
				return true
			}
			if segmentsCollinearOverlap(a, b, c, d) {
				return true
			} // Flags shared edges, ignores shared vertices
		}
	}
	return false
}

func pointInPolygon(px, py float64, poly []GridPoint) bool {
	inside := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if ((poly[i].Y > py) != (poly[j].Y > py)) &&
			(px < (poly[j].X-poly[i].X)*(py-poly[i].Y)/(poly[j].Y-poly[i].Y)+poly[i].X) {
			inside = !inside
		}
	}
	return inside
}
