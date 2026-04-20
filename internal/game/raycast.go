package game

import (
	"fmt"
	"image/color"
	"math"
)

type RaySegment struct {
	Start GridPoint
	End   GridPoint
	Color color.Color
}

type RayResult struct {
	Segments   []RaySegment
	FinalText  string
	FinalColor color.Color
	HitShapes  []*Shape
}

func calculateRayColor(hitColors map[string]bool) (string, color.Color) {
	r, y, b, w := hitColors["red"], hitColors["yellow"], hitColors["blue"], hitColors["white"]

	if r && y && b && w {
		return "Grey", color.RGBA{150, 150, 150, 255}
	}
	if r && y && b {
		return "Black", color.RGBA{0, 0, 0, 255}
	}
	if r && y && w {
		return "Light Orange", color.RGBA{255, 200, 100, 255}
	}
	if r && b && w {
		return "Light Lilla", color.RGBA{200, 150, 255, 255}
	}
	if y && b && w {
		return "Light Green", color.RGBA{150, 255, 150, 255}
	}
	if r && y {
		return "Orange", color.RGBA{255, 165, 0, 255}
	}
	if r && b {
		return "Lilla", color.RGBA{150, 0, 255, 255}
	}
	if y && b {
		return "Green", color.RGBA{0, 255, 0, 255}
	}
	if r && w {
		return "Pink", color.RGBA{255, 192, 203, 255}
	}
	if y && w {
		return "Light Yellow", color.RGBA{255, 255, 150, 255}
	}
	if b && w {
		return "Light Blue", color.RGBA{173, 216, 230, 255}
	}
	if r {
		return "Red", color.RGBA{255, 50, 50, 255}
	}
	if y {
		return "Yellow", color.RGBA{255, 255, 0, 255}
	}
	if b {
		return "Blue", color.RGBA{50, 100, 255, 255}
	}
	if w {
		return "White", color.White
	}

	return "", color.RGBA{0, 255, 255, 255}
}

func getExitLabel(x, y float64) string {
	safeX := max(int(math.Floor(x)), 0)
	if safeX >= cols {
		safeX = cols - 1
	}

	safeY := max(int(math.Floor(y)), 0)
	if safeY >= rows {
		safeY = rows - 1
	}

	if y <= 0.01 {
		return fmt.Sprintf("%d", safeX+1)
	}
	if y >= float64(rows)-0.01 {
		return []string{"i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}[safeX]
	}
	if x <= 0.01 {
		return []string{"a", "b", "c", "d", "e", "f", "g", "h"}[safeY]
	}
	if x >= float64(cols)-0.01 {
		return fmt.Sprintf("%d", safeY+11)
	}
	return "Unknown"
}

func lineIntersect(p, d, a, b GridPoint) (float64, bool, GridPoint) {
	vX, vY := b.X-a.X, b.Y-a.Y
	denom := d.X*vY - d.Y*vX
	if denom == 0 {
		return 0, false, GridPoint{}
	}

	t := ((a.X-p.X)*vY - (a.Y-p.Y)*vX) / denom
	u := ((a.X-p.X)*d.Y - (a.Y-p.Y)*d.X) / denom

	if t > -1e-5 && u >= -1e-5 && u <= 1.00001 {
		return t, true, GridPoint{X: p.X + t*d.X, Y: p.Y + t*d.Y}
	}
	return 0, false, GridPoint{}
}

// Used for Board Validation Line of Sight
func getFirstHit(startX, startY, dirX, dirY float64, shapes []*Shape) *Shape {
	pos := GridPoint{startX - dirX*0.01, startY - dirY*0.01}
	dir := GridPoint{dirX, dirY}
	var closestT float64 = math.MaxFloat64
	var hitShape *Shape

	for _, s := range shapes {
		globalPoints := s.GlobalPoints()
		for i := range globalPoints {
			a := globalPoints[i]
			b := globalPoints[(i+1)%len(globalPoints)]

			t, hit, _ := lineIntersect(pos, dir, a, b)
			if hit && t > 1e-4 && t < closestT {
				closestT = t
				hitShape = s
			}
		}
	}
	return hitShape
}

func fireRay(startX, startY, dirX, dirY float64, shapes []*Shape) *RayResult {
	pos := GridPoint{startX - dirX*0.01, startY - dirY*0.01}
	dir := GridPoint{dirX, dirY}
	hitColors := make(map[string]bool)
	_, currentColor := calculateRayColor(hitColors)

	var segments []RaySegment
	hitSet := make(map[*Shape]bool)

	for range 50 {
		var closestT float64 = math.MaxFloat64
		var hitPoint GridPoint
		var hitNormal GridPoint
		var hitShape *Shape

		for _, s := range shapes {
			globalPoints := s.GlobalPoints()
			for i := range globalPoints {
				a := globalPoints[i]
				b := globalPoints[(i+1)%len(globalPoints)]

				t, hit, pt := lineIntersect(pos, dir, a, b)
				if hit && t > 1e-4 && t < closestT {
					closestT = t
					hitPoint = pt
					hitShape = s

					segDir := GridPoint{b.X - a.X, b.Y - a.Y}
					hitNormal = GridPoint{-segDir.Y, segDir.X}
					length := math.Hypot(hitNormal.X, hitNormal.Y)
					hitNormal.X /= length
					hitNormal.Y /= length
				}
			}
		}

		if hitShape != nil {
			segments = append(segments, RaySegment{Start: pos, End: hitPoint, Color: currentColor})
			hitSet[hitShape] = true

			if hitShape.logicalColor == "black" {
				res := &RayResult{Segments: segments, FinalText: "Absorbed", FinalColor: color.RGBA{100, 100, 100, 255}}
				for s := range hitSet {
					res.HitShapes = append(res.HitShapes, s)
				}
				return res
			}

			if hitShape.logicalColor != "transparent" {
				hitColors[hitShape.logicalColor] = true
			}
			_, currentColor = calculateRayColor(hitColors)

			dot := dir.X*hitNormal.X + dir.Y*hitNormal.Y
			if dot > 0 {
				hitNormal.X = -hitNormal.X
				hitNormal.Y = -hitNormal.Y
				dot = dir.X*hitNormal.X + dir.Y*hitNormal.Y
			}
			dir.X = dir.X - 2*dot*hitNormal.X
			dir.Y = dir.Y - 2*dot*hitNormal.Y

			dir.X = math.Round(dir.X*100) / 100
			dir.Y = math.Round(dir.Y*100) / 100
			pos = hitPoint
		} else {
			var exitT float64 = math.MaxFloat64
			if dir.X > 0 {
				t := (float64(cols) + 0.02 - pos.X) / dir.X
				if t > 0.001 && t < exitT {
					exitT = t
				}
			}
			if dir.X < 0 {
				t := (-0.02 - pos.X) / dir.X
				if t > 0.001 && t < exitT {
					exitT = t
				}
			}
			if dir.Y > 0 {
				t := (float64(rows) + 0.02 - pos.Y) / dir.Y
				if t > 0.001 && t < exitT {
					exitT = t
				}
			}
			if dir.Y < 0 {
				t := (-0.02 - pos.Y) / dir.Y
				if t > 0.001 && t < exitT {
					exitT = t
				}
			}

			finalPt := GridPoint{pos.X + exitT*dir.X, pos.Y + exitT*dir.Y}
			segments = append(segments, RaySegment{Start: pos, End: finalPt, Color: currentColor})

			colName, colVal := calculateRayColor(hitColors)
			exitLabel := getExitLabel(finalPt.X, finalPt.Y)

			res := &RayResult{
				Segments:   segments,
				FinalText:  fmt.Sprintf("Exit: %s\nColor: %s", exitLabel, colName),
				FinalColor: colVal,
			}
			for s := range hitSet {
				res.HitShapes = append(res.HitShapes, s)
			}
			return res
		}
	}
	res := &RayResult{Segments: segments, FinalText: "Trapped in Loop", FinalColor: color.White}
	for s := range hitSet {
		res.HitShapes = append(res.HitShapes, s)
	}
	return res
}
