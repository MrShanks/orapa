package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

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

func New() *Game {
	triIsoPoints := []GridPoint{{0, 2}, {4, 2}, {2, 0}}
	rhombusPoints := []GridPoint{{1, 0}, {0, 1}, {1, 2}, {2, 1}}
	triRightPoints := []GridPoint{{0, 0}, {0, 2}, {2, 2}}
	triSmallIsoPoints := []GridPoint{{0, 1}, {2, 1}, {1, 0}}
	zShapePoints := []GridPoint{{0, 0}, {2, 0}, {3, 1}, {1, 1}}

	f := text.NewGoXFace(basicfont.Face7x13)

	return &Game{
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

	// --- KEYBOARD MOVEMENT ---
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
			}{{1, grid1OffsetX}} // Laser strictly bound to Board 1

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

	titleStr := "ORAPA MINES"
	titleOp := &text.DrawOptions{}
	titleOp.ColorScale.ScaleWithColor(color.White)
	titleOp.PrimaryAlign = text.AlignCenter

	for dx := -1.0; dx <= 1.0; dx += 1.0 {
		for dy := -1.0; dy <= 1.0; dy += 1.0 {
			titleOp.GeoM.Reset()
			titleOp.GeoM.Scale(2.5, 2.5)
			titleOp.GeoM.Translate(ScreenWidth/2+dx, 30+dy)
			text.Draw(screen, titleStr, g.defaultFace, titleOp)
		}
	}

	subOp := &text.DrawOptions{}
	subOp.ColorScale.ScaleWithColor(color.RGBA{200, 200, 100, 255})
	subOp.PrimaryAlign = text.AlignCenter
	subOp.LineSpacing = 18
	subOp.GeoM.Translate(ScreenWidth/2, 75)
	text.Draw(screen, "Opponent puzzle (Left) | Guessing Board (Right)\nDrag shapes into the grid to guess the layout!", g.defaultFace, subOp)

	cmdOp := &text.DrawOptions{}
	cmdOp.ColorScale.ScaleWithColor(color.RGBA{150, 150, 150, 255})
	cmdOp.PrimaryAlign = text.AlignCenter
	cmdOp.GeoM.Translate(ScreenWidth/2, ScreenHeight-25)
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
			resOp.GeoM.Translate(float64(activeOffX)+float64(cols*tileSize)/2, float64(gridOffsetY-90))
			text.Draw(screen, "SCAN REPORT\n-----------\n"+g.lastRay.FinalText, g.defaultFace, resOp)
		}
	}

	// --- DRAW NOTES LOG ---
	notesStartX := float64(grid1OffsetX)
	notesStartY := float64(gridOffsetY + (rows * tileSize) + 80)

	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY-30), float32(notesStartX+410), float32(notesStartY-30), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY+210), float32(notesStartX+410), float32(notesStartY+210), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX-10), float32(notesStartY-30), float32(notesStartX-10), float32(notesStartY+210), 1, gridColor, false)
	vector.StrokeLine(screen, float32(notesStartX+410), float32(notesStartY-30), float32(notesStartX+410), float32(notesStartY+210), 1, gridColor, false)

	titleStr = "Notes (Press ENTER to log note)"
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

func (g *Game) Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }
