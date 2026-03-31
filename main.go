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
	screenWidth  = 800
	screenHeight = 600
	tileSize     = 40
	rows         = 8
	cols         = 10

	// Dynamically center the board based on screen and grid sizes
	gridOffsetX = (screenWidth - (cols * tileSize)) / 2
	gridOffsetY = (screenHeight-(rows*tileSize))/2 + 10
)

type GridPoint struct {
	X, Y float32
}

type Shape struct {
	basePoints    []GridPoint
	localPoints   []GridPoint
	gridX, gridY  int
	clr           color.Color
	rotationSteps int
}

func NewShape(points []GridPoint, x, y int, clr color.Color) *Shape {
	s := &Shape{
		basePoints: points,
		gridX:      x,
		gridY:      y,
		clr:        clr,
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

	finalMinX := s.gridX + int(math.Round(float64(minX)))
	finalMinY := s.gridY + int(math.Round(float64(minY)))
	finalMaxX := s.gridX + int(math.Round(float64(maxX)))
	finalMaxY := s.gridY + int(math.Round(float64(maxY)))

	return finalMinX, finalMinY, finalMaxX, finalMaxY
}

type Game struct {
	shapes        []*Shape
	selectedIndex int
	defaultFace   text.Face
}

func (g *Game) Update() error {
	if len(g.shapes) == 0 {
		return nil
	}
	s := g.shapes[g.selectedIndex]

	oldX, oldY, oldRot := s.gridX, s.gridY, s.rotationSteps
	moved := false
	rotated := false

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

	if moved {
		if rotated {
			minX, minY, maxX, maxY := s.Bounds()
			if maxX > cols {
				s.gridX -= (maxX - cols)
			}
			if maxY > rows {
				s.gridY -= (maxY - rows)
			}
			if minX < 0 {
				s.gridX += (0 - minX)
			}
			if minY < 0 {
				s.gridY += (0 - minY)
			}
		}

		minX, minY, maxX, maxY := s.Bounds()
		valid := true

		if minX < 0 || maxX > cols || minY < 0 || maxY > rows {
			valid = false
		}

		if !valid {
			s.gridX, s.gridY, s.rotationSteps = oldX, oldY, oldRot
			s.applyRotation()
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 55, 65, 255})

	// TITLE
	titleOp := &text.DrawOptions{}
	titleOp.ColorScale.ScaleWithColor(color.White)
	titleOp.PrimaryAlign = text.AlignCenter
	titleOp.GeoM.Scale(2.5, 2.5)
	titleOp.GeoM.Translate(screenWidth/2, 40)
	text.Draw(screen, "ORAPA", g.defaultFace, titleOp)

	// LEGEND
	msgOp := &text.DrawOptions{}
	msgOp.ColorScale.ScaleWithColor(color.RGBA{200, 200, 100, 255})
	msgOp.PrimaryAlign = text.AlignCenter
	msgOp.GeoM.Translate(screenWidth/2, screenHeight-40)
	text.Draw(screen, "Command: Arrows / HJKL (Move) | R (Rotate) | Tab (Switch)", g.defaultFace, msgOp)

	// GRID
	gridColor := color.RGBA{80, 85, 95, 255}
	for i := 0; i <= cols; i++ {
		x := float32(gridOffsetX + (i * tileSize))
		vector.StrokeLine(screen, x, gridOffsetY, x, gridOffsetY+(rows*tileSize), 1, gridColor, false)
	}
	for j := 0; j <= rows; j++ {
		y := float32(gridOffsetY + (j * tileSize))
		vector.StrokeLine(screen, gridOffsetX, y, gridOffsetX+(cols*tileSize), y, 1, gridColor, false)
	}

	// LABELS
	white := color.White
	leftLetters := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	bottomLetters := []string{"i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}

	txtOp := &text.DrawOptions{}
	txtOp.ColorScale.ScaleWithColor(white)

	for i := range cols {
		txtOp.GeoM.Reset()
		txtOp.GeoM.Translate(float64(gridOffsetX+(i*tileSize)+15), float64(gridOffsetY-25))
		text.Draw(screen, fmt.Sprintf("%d", i+1), g.defaultFace, txtOp)

		txtOp.GeoM.Reset()
		txtOp.GeoM.Translate(float64(gridOffsetX+(i*tileSize)+15), float64(gridOffsetY+(rows*tileSize)+10))
		text.Draw(screen, bottomLetters[i], g.defaultFace, txtOp)
	}
	for j := range rows {
		txtOp.GeoM.Reset()
		txtOp.GeoM.Translate(float64(gridOffsetX-30), float64(gridOffsetY+(j*tileSize)+12))
		text.Draw(screen, leftLetters[j], g.defaultFace, txtOp)

		txtOp.GeoM.Reset()
		txtOp.GeoM.Translate(float64(gridOffsetX+(cols*tileSize)+15), float64(gridOffsetY+(j*tileSize)+12))
		text.Draw(screen, fmt.Sprintf("%d", j+11), g.defaultFace, txtOp)
	}

	// SHAPES
	for idx, s := range g.shapes {
		var path vector.Path
		anchorX := float32(gridOffsetX + (s.gridX * tileSize))
		anchorY := float32(gridOffsetY + (s.gridY * tileSize))

		for i, p := range s.localPoints {
			vx, vy := anchorX+(p.X*tileSize), anchorY+(p.Y*tileSize)
			if i == 0 {
				path.MoveTo(vx, vy)
			} else {
				path.LineTo(vx, vy)
			}
		}

		vector.FillPath(screen, &path, &vector.FillOptions{}, &vector.DrawPathOptions{
			ColorScale: func() (cs ebiten.ColorScale) { cs.ScaleWithColor(s.clr); return cs }(),
		})

		// SELECTION HIGHLIGHT
		if idx == g.selectedIndex {
			orange := color.RGBA{255, 165, 0, 255}
			for i := range s.localPoints {
				p1 := s.localPoints[i]
				p2 := s.localPoints[(i+1)%len(s.localPoints)]
				vector.StrokeLine(screen, anchorX+(p1.X*tileSize), anchorY+(p1.Y*tileSize), anchorX+(p2.X*tileSize), anchorY+(p2.Y*tileSize), 3, orange, false)
			}
		}
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
			NewShape(triIsoPoints, 1, 1, color.NRGBA{50, 100, 255, 200}),      // Blue
			NewShape(triIsoPoints, 1, 4, color.NRGBA{255, 255, 255, 230}),     // White
			NewShape(rhombusPoints, 6, 1, color.NRGBA{255, 255, 255, 230}),    // White
			NewShape(triRightPoints, 6, 5, color.NRGBA{255, 255, 0, 200}),     // Yellow
			NewShape(triSmallIsoPoints, 0, 6, color.NRGBA{255, 255, 255, 76}), // Transparent
			NewShape(triSmallIsoPoints, 3, 6, color.NRGBA{0, 0, 0, 200}),      // Black
			NewShape(zShapePoints, 5, 3, color.NRGBA{255, 50, 50, 200}),       // Red
		},
	}

	ebiten.SetWindowTitle("Orapa")
	ebiten.SetWindowSize(screenWidth, screenHeight)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
