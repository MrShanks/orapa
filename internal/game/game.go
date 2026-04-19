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

	rayActive      bool
	rayFrame       int
	rayStartX      float64
	rayStartY      float64
	rayDirX        float64
	rayDirY        float64
	lastRay        *RayResult
	activeRayBoard int

	isDragging     bool
	dragMouseGridX int
	dragMouseGridY int

	notes         []string
	currentNote   []rune
	cursorPos     int
	isTyping      bool
	editingIndex  int
	cursorCounter int

	marks       map[GridPoint]bool
	isMarking   bool
	markingMode bool
	labelMarks  map[string]bool

	resetConfirm  bool
	board1Invalid bool
}

func New() *Game {
	f := text.NewGoXFace(basicfont.Face7x13)
	g := &Game{
		defaultFace:  f,
		marks:        make(map[GridPoint]bool),
		labelMarks:   make(map[string]bool),
		editingIndex: -1,
	}
	g.initShapes()
	return g
}

func (g *Game) initShapes() {
	triIsoPoints := []GridPoint{{0, 2}, {4, 2}, {2, 0}}
	rhombusPoints := []GridPoint{{1, 0}, {0, 1}, {1, 2}, {2, 1}}
	triRightPoints := []GridPoint{{0, 0}, {0, 2}, {2, 2}}
	triSmallIsoPoints := []GridPoint{{0, 1}, {2, 1}, {1, 0}}
	zShapePoints := []GridPoint{{0, 0}, {2, 0}, {3, 1}, {1, 1}}

	g.shapes = []*Shape{
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
	}
}

func validateBoard(shapes []*Shape) bool {
	var b1 []*Shape
	for _, s := range shapes {
		if s.board == 1 {
			b1 = append(b1, s)
		}
	}

	for i := 0; i < len(b1); i++ {
		for j := i + 1; j < len(b1); j++ {
			if shapesOverlapOrEdgeTouch(b1[i], b1[j]) {
				return false
			}
		}
	}

	visibleSet := make(map[*Shape]bool)
	for y := 0.05; y < float64(rows); y += 0.1 {
		if s := getFirstHit(0, y, 1, 0, b1); s != nil {
			visibleSet[s] = true
		}
		if s := getFirstHit(float64(cols), y, -1, 0, b1); s != nil {
			visibleSet[s] = true
		}
	}
	for x := 0.05; x < float64(cols); x += 0.1 {
		if s := getFirstHit(x, 0, 0, 1, b1); s != nil {
			visibleSet[s] = true
		}
		if s := getFirstHit(x, float64(rows), 0, -1, b1); s != nil {
			visibleSet[s] = true
		}
	}

	for _, s := range b1 {
		if !visibleSet[s] {
			return false
		}
	}

	return true
}

func (g *Game) Update() error {
	g.cursorCounter++

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.isTyping {
			finalStr := string(g.currentNote)
			if finalStr != "" {
				if g.editingIndex >= 0 {
					g.notes[g.editingIndex] = finalStr
				} else if len(g.notes) < 30 {
					g.notes = append(g.notes, finalStr)
				}
			} else if g.editingIndex >= 0 {
				g.notes = append(g.notes[:g.editingIndex], g.notes[g.editingIndex+1:]...)
			}
			g.currentNote = nil
			g.isTyping = false
			g.editingIndex = -1
			g.cursorPos = 0
		} else {
			g.isTyping = true
		}
	}

	if g.isTyping {
		chars := ebiten.AppendInputChars(nil)
		if len(chars) > 0 {
			for _, r := range chars {
				g.currentNote = append(g.currentNote[:g.cursorPos], append([]rune{r}, g.currentNote[g.cursorPos:]...)...)
				g.cursorPos++
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if g.cursorPos > 0 {
				g.currentNote = append(g.currentNote[:g.cursorPos-1], g.currentNote[g.cursorPos:]...)
				g.cursorPos--
			} else if len(g.currentNote) == 0 && g.editingIndex == -1 && len(g.notes) > 0 {
				g.notes = g.notes[:len(g.notes)-1]
			}
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && g.cursorPos > 0 {
			g.cursorPos--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) && g.cursorPos < len(g.currentNote) {
			g.cursorPos++
		}
	}

	moved, rotated := false, false
	if !g.isTyping && len(g.shapes) > 0 {
		s := g.shapes[g.selectedIndex]
		oldX, oldY, oldRot, oldFlip := s.gridX, s.gridY, s.rotationSteps, s.flipped

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
		} else if inpututil.IsKeyJustPressed(ebiten.KeyF) {
			s.Flip()
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
				s.gridX, s.gridY, s.rotationSteps, s.flipped = oldX, oldY, oldRot, oldFlip
				s.applyRotation()
			}
		}
	}

	mx, my := ebiten.CursorPosition()
	b2XGrid := int(math.Floor((float64(mx) - float64(grid2OffsetX)) / float64(tileSize)))
	b2YGrid := int(math.Floor((float64(my) - float64(gridOffsetY)) / float64(tileSize)))

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		clickedReset := false
		clickedNote := false

		if mx >= ScreenWidth-150 && mx <= ScreenWidth-20 && my >= 20 && my <= 60 {
			clickedReset = true
			if g.resetConfirm {
				g.initShapes()
				g.notes = nil
				g.marks = make(map[GridPoint]bool)
				g.labelMarks = make(map[string]bool)
				g.rayActive, g.resetConfirm = false, false
			} else {
				g.resetConfirm = true
			}
		}

		notesStartX := float64(grid1OffsetX)
		notesStartY := float64(gridOffsetY + (rows * tileSize) + 80)
		if !clickedReset {
			for i := range g.notes {
				col, row := i/10, i%10
				nx := notesStartX + float64(col*135)
				ny := notesStartY + float64(row*20) + 10
				if float64(mx) >= nx && float64(mx) <= nx+130 && float64(my) >= ny && float64(my) <= ny+20 {
					g.isTyping = true
					g.currentNote = []rune(g.notes[i])
					g.cursorPos = len(g.currentNote)
					g.editingIndex = i
					clickedNote = true
					break
				}
			}
		}

		if !clickedReset && !clickedNote {
			g.resetConfirm = false
			clickedShape := false
			for i := len(g.shapes) - 1; i >= 0; i-- {
				s := g.shapes[i]
				offX := float64(grid1OffsetX)
				if s.board == 2 {
					offX = float64(grid2OffsetX)
				}
				if pointInPolygon((float64(mx)-offX)/float64(tileSize), (float64(my)-gridOffsetY)/float64(tileSize), s.GlobalPoints()) {
					g.selectedIndex, clickedShape, g.isDragging = i, true, true
					g.dragMouseGridX = int(math.Floor((float64(mx) - offX) / float64(tileSize)))
					g.dragMouseGridY = int(math.Floor(float64(my-gridOffsetY) / float64(tileSize)))
					break
				}
			}
			if !clickedShape {
				if b2XGrid >= 0 && b2XGrid < cols && b2YGrid >= 0 && b2YGrid < rows {
					g.isMarking = true
					pt := GridPoint{X: float64(b2XGrid), Y: float64(b2YGrid)}
					g.markingMode = !g.marks[pt]
					g.marks[pt] = g.markingMode
				} else {
					for i := range cols {
						if math.Hypot(float64(mx)-(grid1OffsetX+float64(i*tileSize)+20), float64(my)-(gridOffsetY-15)) < 20 {
							g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(i)+0.5, 0, 0, 1, 1
						}
						if math.Hypot(float64(mx)-(grid1OffsetX+float64(i*tileSize)+20), float64(my)-(gridOffsetY+float64(rows*tileSize)+15)) < 20 {
							g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(i)+0.5, float64(rows), 0, -1, 1
						}
					}
					for j := range rows {
						if math.Hypot(float64(mx)-(grid1OffsetX-15), float64(my)-(gridOffsetY+float64(j*tileSize)+20)) < 20 {
							g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, 0, float64(j)+0.5, 1, 0, 1
						}
						if math.Hypot(float64(mx)-(grid1OffsetX+float64(cols*tileSize)+15), float64(my)-(gridOffsetY+float64(j*tileSize)+20)) < 20 {
							g.rayActive, g.rayFrame, g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, g.activeRayBoard = true, 0, float64(cols), float64(j)+0.5, -1, 0, 1
						}
					}

					leftLetters := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
					bottomLetters := []string{"i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}
					for i := range cols {
						if math.Hypot(float64(mx)-(grid2OffsetX+float64(i*tileSize)+20), float64(my)-(gridOffsetY-15)) < 20 {
							key := fmt.Sprintf("top-%d", i)
							g.labelMarks[key] = !g.labelMarks[key]
						}
						if math.Hypot(float64(mx)-(grid2OffsetX+float64(i*tileSize)+20), float64(my)-(gridOffsetY+float64(rows*tileSize)+15)) < 20 {
							key := "bot-" + bottomLetters[i]
							g.labelMarks[key] = !g.labelMarks[key]
						}
					}
					for j := range rows {
						if math.Hypot(float64(mx)-(grid2OffsetX-15), float64(my)-(gridOffsetY+float64(j*tileSize)+20)) < 20 {
							key := "left-" + leftLetters[j]
							g.labelMarks[key] = !g.labelMarks[key]
						}
						if math.Hypot(float64(mx)-(grid2OffsetX+float64(cols*tileSize)+15), float64(my)-(gridOffsetY+float64(j*tileSize)+20)) < 20 {
							key := fmt.Sprintf("right-%d", j+11)
							g.labelMarks[key] = !g.labelMarks[key]
						}
					}
				}
			}
		}
	}

	if g.isDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s := g.shapes[g.selectedIndex]
		offX := float64(grid1OffsetX)
		if s.board == 2 {
			offX = float64(grid2OffsetX)
		}
		curXGrid := int(math.Floor((float64(mx) - offX) / float64(tileSize)))
		curYGrid := int(math.Floor(float64(my-gridOffsetY) / float64(tileSize)))
		dx, dy := curXGrid-g.dragMouseGridX, curYGrid-g.dragMouseGridY
		if dx != 0 || dy != 0 {
			s.gridX, s.gridY = s.gridX+dx, s.gridY+dy
			if !s.IsValidPosition() {
				s.gridX, s.gridY = s.gridX-dx, s.gridY-dy
			}
			g.dragMouseGridX, g.dragMouseGridY = curXGrid, curYGrid
		}
	} else {
		g.isDragging = false
	}

	if g.isMarking && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if b2XGrid >= 0 && b2XGrid < cols && b2YGrid >= 0 && b2YGrid < rows {
			g.marks[GridPoint{X: float64(b2XGrid), Y: float64(b2YGrid)}] = g.markingMode
		}
	} else {
		g.isMarking = false
	}

	if g.rayActive {
		g.rayFrame++
		var bShapes []*Shape
		for _, s := range g.shapes {
			if s.board == g.activeRayBoard {
				bShapes = append(bShapes, s)
			}
		}
		g.lastRay = fireRay(g.rayStartX, g.rayStartY, g.rayDirX, g.rayDirY, bShapes)
	}

	g.board1Invalid = !validateBoard(g.shapes)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{50, 55, 65, 255})

	btnX, btnY, btnW, btnH := float32(ScreenWidth-150), float32(20), float32(130), float32(40)
	btnClr := color.RGBA{100, 100, 100, 255}
	btnTxt := "RESET BOARD"
	if g.resetConfirm {
		btnClr = color.RGBA{200, 50, 50, 255}
		btnTxt = "SURE?"
	}
	vector.DrawFilledRect(screen, btnX, btnY, btnW, btnH, btnClr, false)
	btnOp := &text.DrawOptions{}
	btnOp.ColorScale.ScaleWithColor(color.White)
	btnOp.PrimaryAlign = text.AlignCenter
	btnOp.GeoM.Translate(float64(btnX+btnW/2), float64(btnY+12))
	text.Draw(screen, btnTxt, g.defaultFace, btnOp)

	titleStr := "ORAPA MINES"
	titleOp := &text.DrawOptions{}
	titleOp.ColorScale.ScaleWithColor(color.White)
	titleOp.PrimaryAlign = text.AlignCenter
	for dx := -1.0; dx <= 1.0; dx += 1.0 {
		for dy := -1.0; dy <= 1.0; dy += 1.0 {
			titleOp.GeoM.Reset()
			titleOp.GeoM.Scale(2.5, 2.5)
			titleOp.GeoM.Translate(float64(ScreenWidth/2)+dx, 30+dy)
			text.Draw(screen, titleStr, g.defaultFace, titleOp)
		}
	}

	subOp := &text.DrawOptions{}
	subOp.ColorScale.ScaleWithColor(color.RGBA{200, 200, 100, 255})
	subOp.PrimaryAlign = text.AlignCenter
	subOp.LineSpacing = 18
	subOp.GeoM.Translate(float64(ScreenWidth/2), 75)
	text.Draw(screen, "Opponent puzzle (Left) | Guessing Board (Right)\nIf Board 1 is RED, your layout is invalid!", g.defaultFace, subOp)

	gridColor1 := color.RGBA{80, 85, 95, 255}
	if g.board1Invalid {
		gridColor1 = color.RGBA{255, 50, 50, 255}
	}
	gridColor2 := color.RGBA{80, 85, 95, 255}

	greyed := color.RGBA{80, 80, 80, 255}
	offsets := []int{grid1OffsetX, grid2OffsetX}
	for _, bOffX := range offsets {
		gClr := gridColor1
		if bOffX == grid2OffsetX {
			gClr = gridColor2
		}

		for i := 0; i <= cols; i++ {
			x := float32(bOffX + (i * tileSize))
			vector.StrokeLine(screen, x, float32(gridOffsetY), x, float32(gridOffsetY+(rows*tileSize)), 1, gClr, false)
		}
		for j := 0; j <= rows; j++ {
			y := float32(gridOffsetY + (j * tileSize))
			vector.StrokeLine(screen, float32(bOffX), y, float32(bOffX+(cols*tileSize)), y, 1, gClr, false)
		}
		leftLetters := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		bottomLetters := []string{"i", "j", "k", "l", "m", "n", "o", "p", "q", "r"}
		txtOp := &text.DrawOptions{}
		txtOp.PrimaryAlign = text.AlignCenter
		for i := range cols {
			lTop, lBot := fmt.Sprintf("%d", i+1), bottomLetters[i]

			txtOp.GeoM.Reset()
			lx, ly := float64(bOffX+(i*tileSize)+20), float64(gridOffsetY-15)
			txtOp.GeoM.Translate(lx, ly)
			txtOp.ColorScale.Reset()
			if bOffX == grid2OffsetX && g.labelMarks[fmt.Sprintf("top-%d", i)] {
				txtOp.ColorScale.ScaleWithColor(greyed)
			} else {
				txtOp.ColorScale.ScaleWithColor(color.White)
			}
			text.Draw(screen, lTop, g.defaultFace, txtOp)

			txtOp.GeoM.Reset()
			lx, ly = float64(bOffX+(i*tileSize)+20), float64(gridOffsetY+(rows*tileSize)+15)
			txtOp.GeoM.Translate(lx, ly)
			txtOp.ColorScale.Reset()
			if bOffX == grid2OffsetX && g.labelMarks["bot-"+lBot] {
				txtOp.ColorScale.ScaleWithColor(greyed)
			} else {
				txtOp.ColorScale.ScaleWithColor(color.White)
			}
			text.Draw(screen, lBot, g.defaultFace, txtOp)
		}
		for j := range rows {
			lLeft, lRight := leftLetters[j], fmt.Sprintf("%d", j+11)

			txtOp.GeoM.Reset()
			lx, ly := float64(bOffX-15), float64(gridOffsetY+(j*tileSize)+20)
			txtOp.GeoM.Translate(lx, ly)
			txtOp.ColorScale.Reset()
			if bOffX == grid2OffsetX && g.labelMarks["left-"+lLeft] {
				txtOp.ColorScale.ScaleWithColor(greyed)
			} else {
				txtOp.ColorScale.ScaleWithColor(color.White)
			}
			text.Draw(screen, lLeft, g.defaultFace, txtOp)

			txtOp.GeoM.Reset()
			lx, ly = float64(bOffX+(cols*tileSize)+15), float64(gridOffsetY+(j*tileSize)+20)
			txtOp.GeoM.Translate(lx, ly)
			txtOp.ColorScale.Reset()
			if bOffX == grid2OffsetX && g.labelMarks["right-"+lRight] {
				txtOp.ColorScale.ScaleWithColor(greyed)
			} else {
				txtOp.ColorScale.ScaleWithColor(color.White)
			}
			text.Draw(screen, lRight, g.defaultFace, txtOp)
		}
	}

	for pt, active := range g.marks {
		if active {
			cx, cy := float32(grid2OffsetX+(int(pt.X)*tileSize)), float32(gridOffsetY+(int(pt.Y)*tileSize))
			vector.StrokeLine(screen, cx+10, cy+10, cx+30, cy+30, 2, color.RGBA{255, 50, 50, 255}, false)
			vector.StrokeLine(screen, cx+30, cy+10, cx+10, cy+30, 2, color.RGBA{255, 50, 50, 255}, false)
		}
	}

	for idx, s := range g.shapes {
		var path vector.Path
		bOffX := float32(grid1OffsetX)
		if s.board == 2 {
			bOffX = float32(grid2OffsetX)
		}
		anchorX, anchorY := bOffX+float32(s.gridX*tileSize), float32(gridOffsetY+s.gridY*tileSize)
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
			for i := range s.localPoints {
				p1, p2 := s.localPoints[i], s.localPoints[(i+1)%len(s.localPoints)]
				vector.StrokeLine(screen, anchorX+float32(p1.X*tileSize), anchorY+float32(p1.Y*tileSize), anchorX+float32(p2.X*tileSize), anchorY+float32(p2.Y*tileSize), 3, color.RGBA{255, 165, 0, 255}, false)
			}
		}
	}

	if g.lastRay != nil {
		raySpeed, drawnDist, totalLen := 0.2, 0.0, 0.0
		currentDist := float64(g.rayFrame) * raySpeed
		activeOffX := float32(grid1OffsetX)
		for _, seg := range g.lastRay.Segments {
			totalLen += math.Hypot(seg.End.X-seg.Start.X, seg.End.Y-seg.Start.Y)
		}
		for _, seg := range g.lastRay.Segments {
			sLen := math.Hypot(seg.End.X-seg.Start.X, seg.End.Y-seg.Start.Y)
			if currentDist > drawnDist {
				drawLen := currentDist - drawnDist
				if drawLen > sLen {
					drawLen = sLen
				}
				ratio := drawLen / sLen
				x1, y1 := activeOffX+float32(seg.Start.X*tileSize), float32(gridOffsetY+seg.Start.Y*tileSize)
				x2 := activeOffX + float32((seg.Start.X+(seg.End.X-seg.Start.X)*ratio)*tileSize)
				y2 := float32(gridOffsetY + (seg.Start.Y+(seg.End.Y-seg.Start.Y)*ratio)*tileSize)
				vector.StrokeLine(screen, x1, y1, x2, y2, 4, seg.Color, false)
			}
			drawnDist += sLen
		}
		if currentDist >= totalLen {
			resOp := &text.DrawOptions{}
			resOp.ColorScale.ScaleWithColor(g.lastRay.FinalColor)
			resOp.PrimaryAlign = text.AlignCenter
			resOp.LineSpacing = 16
			resOp.GeoM.Translate(float64(activeOffX)+float64(cols*tileSize)/2, float64(gridOffsetY-115))
			text.Draw(screen, "SCAN REPORT\n-----------\n"+g.lastRay.FinalText, g.defaultFace, resOp)
		}
	}

	notesStartX, notesStartY := float64(grid1OffsetX), float64(gridOffsetY+(rows*tileSize)+80)
	vector.StrokeRect(screen, float32(notesStartX-10), float32(notesStartY-30), 420, 240, 1, gridColor2, false)

	titleStr = "Notes (Click note to edit | ENTER to save)"
	if g.isTyping {
		before := string(g.currentNote[:g.cursorPos])
		after := string(g.currentNote[g.cursorPos:])
		cursorChar := " "
		if (g.cursorCounter/30)%2 == 0 {
			cursorChar = "|"
		}
		titleStr = "> " + before + cursorChar + after
	}
	tOp := &text.DrawOptions{}
	tOp.ColorScale.ScaleWithColor(color.RGBA{150, 150, 150, 255})
	if g.isTyping {
		tOp.ColorScale.Reset()
		tOp.ColorScale.ScaleWithColor(color.RGBA{255, 165, 0, 255})
	}
	tOp.GeoM.Translate(notesStartX, notesStartY-20)
	text.Draw(screen, titleStr, g.defaultFace, tOp)

	idxColor, contentColor := color.RGBA{60, 60, 60, 80}, color.White
	for i, n := range g.notes {
		col, row := i/10, i%10
		x, y := notesStartX+float64(col*135), notesStartY+float64(row*20)+10
		noteOp := &text.DrawOptions{}
		noteOp.GeoM.Translate(x, y)
		noteOp.ColorScale.ScaleWithColor(idxColor)
		text.Draw(screen, fmt.Sprintf("%d.", i+1), g.defaultFace, noteOp)
		noteOp.GeoM.Translate(25, 0)
		noteOp.ColorScale.Reset()
		noteOp.ColorScale.ScaleWithColor(contentColor)
		text.Draw(screen, n, g.defaultFace, noteOp)
	}

	cmdOp := &text.DrawOptions{}
	cmdOp.ColorScale.ScaleWithColor(color.RGBA{150, 150, 150, 255})
	cmdOp.PrimaryAlign = text.AlignCenter
	cmdOp.GeoM.Translate(float64(ScreenWidth/2), float64(ScreenHeight-25))
	text.Draw(screen, "Move: Drag/HJKL | Rot: R | Flip: F | Tab/Click: Switch | Arrows: Cursor | Note: Enter", g.defaultFace, cmdOp)
}

func (g *Game) Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }
