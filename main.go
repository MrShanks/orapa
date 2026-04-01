package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	screenWidth  = 1050
	screenHeight = 850
	tileSize     = 40
	rows         = 8
	cols         = 10
	grid1OffsetX = 80
	grid2OffsetX = 570
	gridOffsetY  = 150
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

// --- OPTICAL PHYSICS ENGINE ---

type RaySegment struct {
	Start GridPoint
	End   GridPoint
	Color color.Color
}

type RayResult struct {
	Segments   []RaySegment
	FinalText  string
	FinalColor color.Color
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
	safeX := int(math.Floor(x))
	if safeX < 0 {
		safeX = 0
	}
	if safeX >= cols {
		safeX = cols - 1
	}

	safeY := int(math.Floor(y))
	if safeY < 0 {
		safeY = 0
	}
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

	if t > 0.0001 && u >= 0 && u <= 1 {
		return t, true, GridPoint{X: p.X + t*d.X, Y: p.Y + t*d.Y}
	}
	return 0, false, GridPoint{}
}

func fireRay(startX, startY, dirX, dirY float64, shapes []*Shape) *RayResult {
	pos := GridPoint{startX - dirX*0.01, startY - dirY*0.01}
	dir := GridPoint{dirX, dirY}
	hitColors := make(map[string]bool)
	_, currentColor := calculateRayColor(hitColors)

	var segments []RaySegment

	for range 50 {
		var closestT float64 = math.MaxFloat64
		var hitPoint GridPoint
		var hitNormal GridPoint
		var hitShape *Shape

		for _, s := range shapes {
			globalPoints := make([]GridPoint, len(s.localPoints))
			for i, lp := range s.localPoints {
				globalPoints[i] = GridPoint{float64(s.gridX) + lp.X, float64(s.gridY) + lp.Y}
			}

			for i := range globalPoints {
				a := globalPoints[i]
				b := globalPoints[(i+1)%len(globalPoints)]

				t, hit, pt := lineIntersect(pos, dir, a, b)
				if hit && t < closestT {
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

			if hitShape.logicalColor == "black" {
				return &RayResult{Segments: segments, FinalText: "Absorbed", FinalColor: color.RGBA{100, 100, 100, 255}}
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

			return &RayResult{
				Segments:   segments,
				FinalText:  fmt.Sprintf("Exit: %s\nColor: %s", exitLabel, colName),
				FinalColor: colVal,
			}
		}
	}
	return &RayResult{Segments: segments, FinalText: "Trapped in Loop", FinalColor: color.White}
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

// --- MAIN GAME LOGIC ---

type Game struct {
	shapes        []*Shape
	selectedIndex int
	defaultFace   text.Face

	// Laser Ray State
	rayActive      bool
	rayFrame       int
	rayStartX      float64
	rayStartY      float64
	rayDirX        float64
	rayDirY        float64
	lastRay        *RayResult
	activeRayBoard int

	// Mouse Dragging State
	isDragging     bool
	dragMouseGridX int
	dragMouseGridY int

	// Intel Notes State
	notes       []string
	currentNote string
	isTyping    bool
}

func (g *Game) Update() error {
	// --- TYPING LOGIC ---
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.isTyping {
			if g.currentNote != "" && len(g.notes) < 30 {
				g.notes = append(g.notes, g.currentNote)
			}
			g.currentNote = ""
			g.isTyping = false
		} else {
			g.isTyping = true
		}
	}

	if g.isTyping {
		g.currentNote += string(ebiten.InputChars())
		runes := []rune(g.currentNote)
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if len(runes) > 0 {
				g.currentNote = string(runes[:len(runes)-1])
			} else if len(g.notes) > 0 {
				g.notes = g.notes[:len(g.notes)-1]
			}
		}
	}

	// --- KEYBOARD MOVEMENT (Only if NOT typing!) ---
	moved, rotated := false, false
	s := g.shapes[g.selectedIndex]
	oldX, oldY, oldRot := s.gridX, s.gridY, s.rotationSteps

	if !g.isTyping && len(g.shapes) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
			s.gridX--
			moved = true
		} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyL) {
			s.gridX++
			moved = true
		} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
			s.gridY--
			moved = true
		} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
			s.gridY++
			moved = true
		} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			s.Rotate()
			moved = true
			rotated = true
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
			g.selectedIndex = (g.selectedIndex + 1) % len(g.shapes)
		}
	}

	if moved {
		if rotated {
			minX, minY, maxX, maxY := s.Bounds()
			if maxX > cols {
				s.gridX -= (maxX - cols)
			}
			if s.board == 1 && maxY > rows {
				s.gridY -= (maxY - rows)
			}
			if s.board == 2 && maxY > rows+8 {
				s.gridY -= (maxY - (rows + 8))
			}
			if minX < 0 {
				s.gridX += (0 - minX)
			}
			if minY < 0 {
				s.gridY += (0 - minY)
			}
		}

		if !s.IsValidPosition() {
			s.gridX, s.gridY, s.rotationSteps = oldX, oldY, oldRot
			s.applyRotation()
		}
	}

	// --- MOUSE LOGIC ---
	mx, my := ebiten.CursorPosition()

	offsetX := float64(grid1OffsetX)
	if len(g.shapes) > 0 && g.shapes[g.selectedIndex].board == 2 {
		offsetX = float64(grid2OffsetX)
	}
	mouseXGrid := int(math.Floor((float64(mx) - offsetX) / float64(tileSize)))
	mouseYGrid := int(math.Floor(float64(my-gridOffsetY) / float64(tileSize)))

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		clickedFired := false
		clickedShape := false

		for i := len(g.shapes) - 1; i >= 0; i-- {
			shape := g.shapes[i]
			shapeOffsetX := float64(grid1OffsetX)
			if shape.board == 2 {
				shapeOffsetX = float64(grid2OffsetX)
			}

			floatMouseXGrid := (float64(mx) - shapeOffsetX) / float64(tileSize)
			floatMouseYGrid := float64(my-gridOffsetY) / float64(tileSize)

			globalPoints := make([]GridPoint, len(shape.localPoints))
			for j, lp := range shape.localPoints {
				globalPoints[j] = GridPoint{float64(shape.gridX) + lp.X, float64(shape.gridY) + lp.Y}
			}

			if pointInPolygon(floatMouseXGrid, floatMouseYGrid, globalPoints) {
				g.selectedIndex = i
				clickedShape = true
				g.isDragging = true

				if g.shapes[g.selectedIndex].board == 2 {
					g.dragMouseGridX = int(math.Floor((float64(mx) - float64(grid2OffsetX)) / float64(tileSize)))
				} else {
					g.dragMouseGridX = int(math.Floor((float64(mx) - float64(grid1OffsetX)) / float64(tileSize)))
				}
				g.dragMouseGridY = mouseYGrid
				break
			}
		}

		if !clickedShape {
			offsets := []struct {
				id int
				x  int
			}{{1, grid1OffsetX}, {2, grid2OffsetX}}

			for _, bd := range offsets {
				bOffX := bd.x

				for i := range cols {
					lx, ly := bOffX+(i*tileSize)+20, gridOffsetY-15
					if math.Hypot(float64(mx-lx), float64(my-ly)) < 20 {
						g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(i)+0.5, 0, 0, 1, bd.id
						clickedFired = true
					}
					lx, ly = bOffX+(i*tileSize)+20, gridOffsetY+(rows*tileSize)+15
					if math.Hypot(float64(mx-lx), float64(my-ly)) < 20 {
						g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(i)+0.5, float64(rows), 0, -1, bd.id
						clickedFired = true
					}
				}
				for j := range rows {
					lx, ly := bOffX-15, gridOffsetY+(j*tileSize)+20
					if math.Hypot(float64(mx-lx), float64(my-ly)) < 20 {
						g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, 0, float64(j)+0.5, 1, 0, bd.id
						clickedFired = true
					}
					lx, ly = bOffX+(cols*tileSize)+15, gridOffsetY+(j*tileSize)+20
					if math.Hypot(float64(mx-lx), float64(my-ly)) < 20 {
						g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(cols), float64(j)+0.5, -1, 0, bd.id
						clickedFired = true
					}
				}
			}
			if !clickedFired {
				g.rayActive = false
			}
		}
	}

	if g.isDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			deltaX := mouseXGrid - g.dragMouseGridX
			deltaY := mouseYGrid - g.dragMouseGridY

			if deltaX != 0 || deltaY != 0 {
				activeShape := g.shapes[g.selectedIndex]
				activeShape.gridX += deltaX
				activeShape.gridY += deltaY

				if !activeShape.IsValidPosition() {
					activeShape.gridX -= deltaX
					activeShape.gridY -= deltaY
				}

				g.dragMouseGridX = mouseXGrid
				g.dragMouseGridY = mouseYGrid
			}
		} else {
			g.isDragging = false
		}
	}

	if g.rayActive {
		g.rayFrame++

		boardShapes := make([]*Shape, 0)
		for _, shape := range g.shapes {
			if shape.board == g.activeRayBoard {
				boardShapes = append(boardShapes, shape)
			}
		}

		g.lastRay = fireRay(g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, boardShapes)
	} else {
		g.lastRay = nil
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 55, 65, 255})

	titleOp := &text.DrawOptions{}
	titleOp.ColorScale.ScaleWithColor(color.White)
	titleOp.PrimaryAlign = text.AlignCenter
	titleOp.GeoM.Scale(2.5, 2.5)
	titleOp.GeoM.Translate(screenWidth/2, 35)
	text.Draw(screen, "ORAPA MINES", g.defaultFace, titleOp)

	subOp := &text.DrawOptions{}
	subOp.ColorScale.ScaleWithColor(color.RGBA{200, 200, 100, 255})
	subOp.PrimaryAlign = text.AlignCenter
	subOp.LineSpacing = 18
	subOp.GeoM.Translate(screenWidth/2, 75)
	text.Draw(screen, "Opponent puzzle (Left) | Guessing Board (Right)\nDrag shapes into the grid to guess the layout!", g.defaultFace, subOp)

	cmdOp := &text.DrawOptions{}
	cmdOp.ColorScale.ScaleWithColor(color.RGBA{150, 150, 150, 255})
	cmdOp.PrimaryAlign = text.AlignCenter
	cmdOp.GeoM.Translate(screenWidth/2, screenHeight-25)
	text.Draw(screen, "Move: Drag/HJKL | Rotate: R | Switch: Tab/Click | Fire Ray: Click Labels | Note: Enter", g.defaultFace, cmdOp)

	gridColor := color.RGBA{80, 85, 95, 255}
	offsets := []int{grid1OffsetX, grid2OffsetX}

	for _, bOffX := range offsets {
		for i := 0; i <= cols; i++ {
			x := float32(bOffX + (i * tileSize))
			vector.StrokeLine(screen, x, float32(gridOffsetY), x, float32(gridOffsetY+(rows*tileSize)), 1, gridColor, false)
		}
		for j := 0; j <= rows; j++ {
			y := float32(gridOffsetY + (j * tileSize))
			vector.StrokeLine(screen, float32(bOffX), y, float32(bOffX+(cols*tileSize)), y, 1, gridColor, false)
		}

		white := color.White
		leftLetters := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		bottomLetters := []string{"i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}

		txtOp := &text.DrawOptions{}
		txtOp.ColorScale.ScaleWithColor(white)
		txtOp.PrimaryAlign = text.AlignCenter

		for i := range cols {
			txtOp.GeoM.Reset()
			txtOp.GeoM.Translate(float64(bOffX+(i*tileSize)+20), float64(gridOffsetY-15))
			text.Draw(screen, fmt.Sprintf("%d", i+1), g.defaultFace, txtOp)

			txtOp.GeoM.Reset()
			txtOp.GeoM.Translate(float64(bOffX+(i*tileSize)+20), float64(gridOffsetY+(rows*tileSize)+15))
			text.Draw(screen, bottomLetters[i], g.defaultFace, txtOp)
		}
		for j := range rows {
			txtOp.GeoM.Reset()
			txtOp.GeoM.Translate(float64(bOffX-15), float64(gridOffsetY+(j*tileSize)+20))
			text.Draw(screen, leftLetters[j], g.defaultFace, txtOp)

			txtOp.GeoM.Reset()
			txtOp.GeoM.Translate(float64(bOffX+(cols*tileSize)+15), float64(gridOffsetY+(j*tileSize)+20))
			text.Draw(screen, fmt.Sprintf("%d", j+11), g.defaultFace, txtOp)
		}
	}

	for idx, s := range g.shapes {
		var path vector.Path

		bOffX := grid1OffsetX
		if s.board == 2 {
			bOffX = grid2OffsetX
		}

		anchorX := float32(bOffX + (s.gridX * tileSize))
		anchorY := float32(gridOffsetY + (s.gridY * tileSize))

		for i, p := range s.localPoints {
			vx, vy := anchorX+float32(p.X*tileSize), anchorY+float32(p.Y*tileSize)
			if i == 0 {
				path.MoveTo(vx, vy)
			} else {
				path.LineTo(vx, vy)
			}
		}

		vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{
			ColorScale: func() (cs ebiten.ColorScale) { cs.ScaleWithColor(s.clr); return cs }(),
		})

		if idx == g.selectedIndex {
			orange := color.RGBA{255, 165, 0, 255}
			for i := range s.localPoints {
				p1 := s.localPoints[i]
				p2 := s.localPoints[(i+1)%len(s.localPoints)]
				vector.StrokeLine(screen, anchorX+float32(p1.X*tileSize), anchorY+float32(p1.Y*tileSize), anchorX+float32(p2.X*tileSize), anchorY+float32(p2.Y*tileSize), 3, orange, false)
			}
		}
	}

	if g.lastRay != nil {
		raySpeed := 0.2
		currentDist := float64(g.rayFrame) * raySpeed
		drawnDist := 0.0
		totalLen := 0.0

		activeOffX := float32(grid1OffsetX)
		if g.activeRayBoard == 2 {
			activeOffX = float32(grid2OffsetX)
		}

		for _, seg := range g.lastRay.Segments {
			segLen := math.Hypot(seg.End.X-seg.Start.X, seg.End.Y-seg.Start.Y)
			totalLen += segLen

			if currentDist <= drawnDist {
				continue
			}

			drawLen := currentDist - drawnDist
			if drawLen > segLen {
				drawLen = segLen
			}

			ratio := drawLen / segLen

			x1, y1 := activeOffX+float32(seg.Start.X*tileSize), float32(gridOffsetY+seg.Start.Y*tileSize)
			x2 := activeOffX + float32((seg.Start.X+(seg.End.X-seg.Start.X)*ratio)*tileSize)
			y2 := float32(gridOffsetY + (seg.Start.Y+(seg.End.Y-seg.Start.Y)*ratio)*tileSize)

			vector.StrokeLine(screen, x1, y1, x2, y2, 4, seg.Color, false)
			drawnDist += segLen
		}

		if currentDist >= totalLen && len(g.lastRay.Segments) > 0 {
			resOp := &text.DrawOptions{}
			resOp.ColorScale.ScaleWithColor(g.lastRay.FinalColor)
			resOp.LineSpacing = 16
			resOp.PrimaryAlign = text.AlignCenter
			resOp.GeoM.Translate(float64(activeOffX)+float64(cols*tileSize)/2, float64(gridOffsetY-55))
			text.Draw(screen, "SCAN REPORT\n-----------\n"+g.lastRay.FinalText, g.defaultFace, resOp)
		}
	}

	// --- DRAW INTELLIGENCE LOG (Notes) ---
	notesStartX := float64(grid1OffsetX)

	notesStartY := float64(gridOffsetY + (rows * tileSize) + 80)

	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY-30), float32(notesStartX+410), float32(notesStartY-30), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY+210), float32(notesStartX+410), float32(notesStartY+210), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY-30), float32(notesStartX-10), float32(notesStartY+210), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX+410), float32(notesStartY-30), float32(notesStartX+410), float32(notesStartY+210), 1, gridColor, false)

	titleStr := "Notes (Press ENTER to log note)"
	if g.isTyping {
		titleStr = "TYPING... (Press ENTER to save) > " + g.currentNote + "_"
	}

	noteTitleOp := &text.DrawOptions{}
	if g.isTyping {
		noteTitleOp.ColorScale.ScaleWithColor(color.RGBA{255, 165, 0, 255})
	} else {
		noteTitleOp.ColorScale.ScaleWithColor(color.RGBA{150, 150, 150, 255})
	}
	noteTitleOp.GeoM.Translate(notesStartX, notesStartY-20)
	text.Draw(screen, titleStr, g.defaultFace, noteTitleOp)

	noteOp := &text.DrawOptions{}
	noteOp.ColorScale.ScaleWithColor(color.White)

	for i, n := range g.notes {
		col := i / 10
		row := i % 10

		x := notesStartX + float64(col*135)
		y := notesStartY + float64(row*20) + 10

		noteOp.GeoM.Reset()
		noteOp.GeoM.Translate(x, y)
		text.Draw(screen, fmt.Sprintf("%d. %s", i+1, n), g.defaultFace, noteOp)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return screenWidth, screenHeight }

func main() {
	triIsoPoints := []GridPoint{{0, 2}, {4, 2}, {2, 0}}
	rhombusPoints := []GridPoint{{1, 0}, {0, 1}, {1, 2}, {2, 1}}
	triRightPoints := []GridPoint{{0, 0}, {0, 2}, {2, 2}}
	triSmallIsoPoints := []GridPoint{{0, 1}, {2, 1}, {1, 0}}
	zShapePoints := []GridPoint{{0, 0}, {2, 0}, {3, 1}, {1, 1}}

	f := text.NewGoXFace(basicfont.Face7x13)

	game := &Game{
		defaultFace: f,
		shapes: []*Shape{
			NewShape(triIsoPoints, 1, 1, color.NRGBA{50, 100, 255, 200}, "blue", 1),
			NewShape(triIsoPoints, 1, 4, color.NRGBA{255, 255, 255, 240}, "white", 1),
			NewShape(rhombusPoints, 6, 1, color.NRGBA{255, 255, 240, 240}, "white", 1),
			NewShape(triRightPoints, 6, 5, color.NRGBA{255, 255, 0, 200}, "yellow", 1),
			NewShape(triSmallIsoPoints, 0, 6, color.NRGBA{255, 255, 255, 50}, "transparent", 1),
			NewShape(triSmallIsoPoints, 3, 6, color.NRGBA{0, 0, 0, 200}, "black", 1),
			NewShape(zShapePoints, 5, 3, color.NRGBA{255, 50, 50, 200}, "red", 1),

			NewShape(triIsoPoints, 0, 9, color.NRGBA{50, 100, 255, 200}, "blue", 2),
			NewShape(triIsoPoints, 5, 9, color.NRGBA{255, 255, 255, 240}, "white", 2),
			NewShape(triRightPoints, 0, 12, color.NRGBA{255, 255, 0, 200}, "yellow", 2),
			NewShape(rhombusPoints, 3, 12, color.NRGBA{255, 255, 255, 240}, "white", 2),
			NewShape(zShapePoints, 6, 12, color.NRGBA{255, 50, 50, 200}, "red", 2),
			NewShape(triSmallIsoPoints, 6, 14, color.NRGBA{255, 255, 255, 50}, "transparent", 2),
			NewShape(triSmallIsoPoints, 3, 14, color.NRGBA{0, 0, 0, 200}, "black", 2),
		},
	}

	ebiten.SetWindowTitle("Orapa Mine")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
