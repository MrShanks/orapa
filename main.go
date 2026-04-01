package main

import (
	"log"

	"github.com/MrShanks/orapa/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := game.New()

	ebiten.SetWindowTitle("Orapa Online")
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
