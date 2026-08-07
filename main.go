package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

const tile = 32
const version = "v0.1.1"

type Node struct {
	x, y int
	kind string
	used bool
}
type Animal struct {
	x, y, vx, vy float64
	active       bool
	turnIn       float64
}
type Game struct {
	day                        int
	hour                       float64
	tickTimer                  float64
	hunger, warmth, wood, food int
	shelter, fire              bool
	weather, message           string
	x, y                       float64
	nodes                      []Node
	animals                    []Animal
	rng                        *rand.Rand
}

func NewGame() *Game {
	return &Game{
		day: 1, warmth: 70, hunger: 75, wood: 3, food: 2, weather: "clear",
		x: 320, y: 250, rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		message: "Explore the island. Static resources are guaranteed; moving game is not.",
		nodes:   []Node{{110, 120, "wood", false}, {170, 260, "wood", false}, {500, 150, "wood", false}, {535, 255, "wood", false}, {95, 305, "wood", false}, {350, 105, "wood", false}, {420, 210, "wood", false}, {450, 285, "food", false}, {260, 245, "camp", false}},
		animals: []Animal{{390, 120, .5, .2, true, 1}, {520, 300, -.3, .4, true, 2}, {210, 170, .2, -.4, true, 3}},
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*g = *NewGame()
		return nil
	}
	dx, dy := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		dx--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		dx++
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		dy--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		dy++
	}
	g.x += dx * 2.2
	g.y += dy * 2.2
	if g.x < 48 {
		g.x = 48
	}
	if g.x > 592 {
		g.x = 592
	}
	if g.y < 62 {
		g.y = 62
	}
	if g.y > 365 {
		g.y = 365
	}
	g.updateAnimals()
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.gather()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.fish()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.hunt()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && g.near("camp") && g.wood >= 2 {
		g.wood -= 2
		g.fire = true
		g.message = "The fire catches. Warmth returns."
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyB) && g.near("camp") && !g.shelter && g.wood >= 6 {
		g.wood -= 6
		g.shelter = true
		g.message = "A rough shelter stands against the wind."
	}
	g.hour += 1.0 / 60.0
	if g.hour >= 24 {
		g.day++
		g.hour = 0
		g.weather = []string{"clear", "wind", "rain"}[g.rng.Intn(3)]
	}
	g.tickTimer += 1.0 / 60.0
	if g.tickTimer >= 1 {
		g.tickTimer -= 1
		g.tick()
	}
	return nil
}

func (g *Game) updateAnimals() {
	for i := range g.animals {
		a := &g.animals[i]
		if !a.active {
			continue
		}
		a.x += a.vx
		a.y += a.vy
		a.turnIn -= 1.0 / 60.0
		if a.x < 70 || a.x > 570 {
			a.vx *= -1
			a.x += a.vx
		}
		if a.y < 80 || a.y > 335 {
			a.vy *= -1
			a.y += a.vy
		}
		if a.turnIn <= 0 {
			a.vx = (g.rng.Float64() - .5) * 1.2
			a.vy = (g.rng.Float64() - .5) * 1.2
			a.turnIn = 1 + g.rng.Float64()*3
		}
	}
}
func (g *Game) gather() {
	for i := range g.nodes {
		n := &g.nodes[i]
		if n.used || abs(float64(n.x)-g.x) >= 42 || abs(float64(n.y)-g.y) >= 42 {
			continue
		}
		if n.kind == "wood" && g.wood < 12 {
			g.wood++
			n.used = true
			g.message = "You gather dry wood. No luck involved."
			return
		}
		if n.kind == "food" {
			g.food++
			n.used = true
			g.message = "You gather an edible plant. No luck involved."
			return
		}
	}
	g.message = "Nothing useful is close enough to gather."
}
func (g *Game) fish() {
	if !g.nearWater() {
		g.message = "You need to stand at the shoreline to fish."
		return
	}
	if g.rng.Float64() < .55 {
		g.food++
		g.message = "A fish bites. Dinner is secured."
	} else {
		g.message = "The line goes slack. Nothing bites."
	}
}
func (g *Game) hunt() {
	for i := range g.animals {
		a := &g.animals[i]
		if !a.active || abs(a.x-g.x) >= 34 || abs(a.y-g.y) >= 34 {
			continue
		}
		if g.rng.Float64() < .5 {
			a.active = false
			g.food++
			g.message = "The hunt succeeds. You bring food back to camp."
		} else {
			a.vx *= 3
			a.vy *= 3
			a.turnIn = 2
			g.message = "The animal bolts. The hunt got away."
		}
		return
	}
	g.message = "No moving game is close enough to hunt."
}
func (g *Game) near(kind string) bool {
	for _, n := range g.nodes {
		if n.kind == kind && abs(float64(n.x)-g.x) < 42 && abs(float64(n.y)-g.y) < 42 {
			return true
		}
	}
	return false
}
func (g *Game) nearWater() bool { return g.y < 88 || g.y > 335 }
func (g *Game) canGather() bool {
	for _, n := range g.nodes {
		if n.used || abs(float64(n.x)-g.x) >= 42 || abs(float64(n.y)-g.y) >= 42 {
			continue
		}
		if n.kind == "food" || (n.kind == "wood" && g.wood < 12) {
			return true
		}
	}
	return false
}
func (g *Game) canHunt() bool {
	for _, a := range g.animals {
		if a.active && abs(a.x-g.x) < 34 && abs(a.y-g.y) < 34 {
			return true
		}
	}
	return false
}
func (g *Game) actionColor(available bool) color.Color {
	if available {
		return color.RGBA{120, 235, 145, 255}
	}
	return color.RGBA{235, 105, 105, 255}
}
func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
func (g *Game) tick() {
	if g.hunger <= 35 && g.food > 0 {
		g.food--
		g.hunger += 24
		g.message = "Hunger triggers an automatic meal."
	}
	g.hunger--
	if g.fire {
		g.warmth += 2
	} else {
		g.warmth--
	}
	if g.weather == "wind" {
		g.warmth--
	}
	if g.weather == "rain" && !g.shelter {
		g.warmth -= 2
	}
	if g.hunger <= 0 || g.warmth <= 0 {
		g.message = "You did not make it. Press R to try again."
	}
	if g.hunger < 0 {
		g.hunger = 0
	}
	if g.warmth < 0 {
		g.warmth = 0
	}
	if g.warmth > 100 {
		g.warmth = 100
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{22, 31, 35, 255})
	ebitenutil.DrawRect(screen, 32, 48, 576, 320, color.RGBA{53, 91, 72, 255})
	for y := 64; y < 368; y += tile {
		for x := 32; x < 608; x += tile {
			c := color.RGBA{58, 98, 73, 255}
			if (x/tile+y/tile)%3 == 0 {
				c = color.RGBA{61, 103, 76, 255}
			}
			ebitenutil.DrawRect(screen, float64(x), float64(y), tile-1, tile-1, c)
		}
	}
	ebitenutil.DrawRect(screen, 32, 48, 576, 16, color.RGBA{72, 126, 143, 255})
	ebitenutil.DrawRect(screen, 32, 352, 576, 16, color.RGBA{72, 126, 143, 255})
	for _, n := range g.nodes {
		if n.used {
			continue
		}
		if n.kind == "wood" && g.wood < 12 {
			ebitenutil.DrawRect(screen, float64(n.x-8), float64(n.y-12), 16, 24, color.RGBA{38, 64, 40, 255})
			ebitenutil.DrawRect(screen, float64(n.x-13), float64(n.y-5), 26, 7, color.RGBA{102, 67, 41, 255})
		}
		if n.kind == "food" {
			ebitenutil.DrawRect(screen, float64(n.x-8), float64(n.y-8), 16, 16, color.RGBA{180, 145, 71, 255})
		}
		if n.kind == "camp" {
			ebitenutil.DrawRect(screen, float64(n.x-16), float64(n.y-10), 32, 20, color.RGBA{93, 59, 43, 255})
		}
	}
	for _, a := range g.animals {
		if a.active {
			ebitenutil.DrawRect(screen, a.x-7, a.y-5, 14, 10, color.RGBA{145, 94, 54, 255})
			ebitenutil.DrawRect(screen, a.x+4, a.y-8, 6, 6, color.RGBA{175, 120, 70, 255})
		}
	}
	if g.shelter {
		ebitenutil.DrawRect(screen, 230, 205, 62, 38, color.RGBA{91, 57, 43, 255})
	}
	if g.fire {
		ebitenutil.DrawRect(screen, 247, 226, 28, 14, color.RGBA{235, 126, 43, 255})
	}
	ebitenutil.DrawRect(screen, g.x-8, g.y-8, 16, 16, color.RGBA{231, 194, 142, 255})
	ebitenutil.DrawRect(screen, g.x-5, g.y-13, 10, 7, color.RGBA{85, 52, 36, 255})
	text.Draw(screen, fmt.Sprintf("LAST LIGHT   DAY %d   %02d:00   %s", g.day, int(g.hour), g.weather), basicfont.Face7x13, 32, 28, color.White)
	text.Draw(screen, version, basicfont.Face7x13, 575, 28, color.RGBA{180, 205, 190, 255})
	text.Draw(screen, fmt.Sprintf("HUNGER %d   WARMTH %d   WOOD %d   FOOD %d", g.hunger, g.warmth, g.wood, g.food), basicfont.Face7x13, 32, 397, color.White)
	text.Draw(screen, "WASD / ARROWS move", basicfont.Face7x13, 32, 420, color.White)
	text.Draw(screen, "E gather", basicfont.Face7x13, 190, 420, g.actionColor(g.canGather()))
	text.Draw(screen, "F fire", basicfont.Face7x13, 260, 420, g.actionColor(g.near("camp") && g.wood >= 2))
	text.Draw(screen, "B shelter", basicfont.Face7x13, 315, 420, g.actionColor(g.near("camp") && !g.shelter && g.wood >= 6))
	text.Draw(screen, "G fish", basicfont.Face7x13, 32, 442, g.actionColor(g.nearWater()))
	text.Draw(screen, "R hunt", basicfont.Face7x13, 90, 442, g.actionColor(g.canHunt()))
	text.Draw(screen, "ESC restart", basicfont.Face7x13, 155, 442, color.White)
	text.Draw(screen, g.message, basicfont.Face7x13, 32, 464, color.RGBA{240, 220, 160, 255})
}
func (g *Game) Layout(_, _ int) (int, int) { return 640, 480 }
func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Last Light")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
