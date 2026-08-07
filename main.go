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
const version = "v0.6.0"

type Food struct {
	name                                 string
	calories, protein, carbs, fat, fiber int
	risk                                 float64
}
type Nutrition struct{ calories, protein, carbs, fat, fiber int }
type Node struct {
	x, y int
	kind string
	used bool
	food Food
}
type Animal struct {
	x, y, vx, vy float64
	active       bool
	turnIn       float64
}
type Game struct {
	day                        int
	hour, tickTimer, moveTimer float64
	hunger, warmth, wood       int
	shelter, fire              bool
	weather, message           string
	x, y                       float64
	nodes                      []Node
	animals                    []Animal
	foods                      []Food
	rng                        *rand.Rand
	nutrition                  Nutrition
	sickTimer                  int
	scores                     []Score
	dead                       bool
	events                     []LogEvent
}

func NewGame() *Game {
	berries := Food{"shore berries", 120, 1, 28, 0, 4, 0}
	return &Game{day: 1, warmth: 70, hunger: 75, wood: 3, weather: "clear", x: 320, y: 250, rng: rand.New(rand.NewSource(time.Now().UnixNano())), message: "Explore the island. Q interacts with whatever is closest.", foods: []Food{berries, berries}, nodes: []Node{
		{110, 120, "wood", false, Food{}}, {170, 260, "wood", false, Food{}}, {500, 150, "wood", false, Food{}}, {535, 255, "wood", false, Food{}}, {95, 305, "wood", false, Food{}}, {350, 105, "wood", false, Food{}}, {420, 210, "wood", false, Food{}},
		{450, 285, "plant", false, Food{"shore berries", 120, 1, 28, 0, 4, 0}}, {145, 185, "plant", false, Food{"wild greens", 80, 3, 10, 0, 5, 0}}, {330, 125, "plant", false, Food{"edible root", 320, 4, 72, 1, 8, 0}}, {470, 230, "plant", false, Food{"questionable mushroom", 60, 2, 8, 1, 2, .35}}, {260, 245, "camp", false, Food{}},
	}, animals: []Animal{{390, 120, .5, .2, true, 1}, {520, 300, -.3, .4, true, 2}, {210, 170, .2, -.4, true, 3}}, scores: loadScores()}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*g = *NewGame()
		return nil
	}
	if g.dead {
		return nil
	}
	if len(g.events) == 0 {
		g.log("run_start", "new survival run")
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
	if dx != 0 || dy != 0 {
		g.moveTimer += 1.0 / 60.0
		if g.moveTimer >= 1 {
			g.moveTimer = 0
			g.expend(35, 1, 5, 1)
			g.log("movement", "movement expenditure")
		}
	}
	g.updateAnimals()
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.interact()
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) && g.near("camp") && g.wood >= 2 {
		g.expend(30, 0, 5, 1)
		g.wood -= 2
		g.fire = true
		g.log("fire", "lit fire at camp")
		g.message = "The fire catches. Warmth returns."
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) && g.near("camp") && !g.shelter && g.wood >= 6 {
		g.expend(60, 0, 8, 2)
		g.wood -= 6
		g.shelter = true
		g.log("shelter", "built shelter at camp")
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
func (g *Game) interact() {
	if g.canGather() {
		g.gather()
		return
	}
	if g.canHunt() {
		g.hunt()
		return
	}
	if g.nearWater() {
		g.fish()
		return
	}
	g.message = "Nothing useful is close enough to interact with."
}
func (g *Game) gather() {
	for i := range g.nodes {
		n := &g.nodes[i]
		if n.used || abs(float64(n.x)-g.x) >= 42 || abs(float64(n.y)-g.y) >= 42 {
			continue
		}
		if n.kind == "wood" && g.wood < 12 {
			g.expend(20, 1, 3, 0)
			g.wood++
			n.used = true
			g.message = "You gather dry wood. No luck involved."
			return
		}
		if n.kind == "plant" {
			g.expend(15, 0, 2, 0)
			g.foods = append(g.foods, n.food)
			n.used = true
			g.message = fmt.Sprintf("You gather %s. It will be eaten automatically when hunger is low.", n.food.name)
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
	g.expend(35, 1, 5, 1)
	g.log("movement", "movement expenditure")
	if g.rng.Float64() < .55 {
		g.foods = append(g.foods, Food{"fish", 500, 35, 0, 28, 0, 0})
		g.log("fish_success", "caught fish")
		g.message = "A fish bites. A protein-rich meal is secured."
	} else {
		g.log("fish_failure", "no fish caught")
		g.message = "The line goes slack. Nothing bites."
	}
}
func (g *Game) hunt() {
	for i := range g.animals {
		a := &g.animals[i]
		if !a.active || abs(a.x-g.x) >= 34 || abs(a.y-g.y) >= 34 {
			continue
		}
		g.expend(60, 2, 6, 2)
		if g.rng.Float64() < .5 {
			a.active = false
			g.foods = append(g.foods, Food{"game meat", 600, 45, 0, 20, 0, 0})
			g.log("hunt_success", "caught game")
			g.message = "The hunt succeeds. You bring nutrient-dense meat back to camp."
		} else {
			a.vx *= 3
			a.vy *= 3
			a.turnIn = 2
			g.log("hunt_failure", "game escaped")
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
		if n.kind == "plant" || (n.kind == "wood" && g.wood < 12) {
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
func (g *Game) interactionLabel() (string, color.Color) {
	if g.canGather() {
		return "Q gather", g.actionColor(true)
	}
	if g.canHunt() {
		return "Q hunt", g.actionColor(true)
	}
	if g.nearWater() {
		return "Q fish", g.actionColor(true)
	}
	return "Q interact", g.actionColor(false)
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
func (g *Game) expend(calories, protein, carbs, fat int) {
	if g.nutrition.calories < calories {
		g.hunger--
		if g.hunger < 0 {
			g.hunger = 0
		}
	}
	g.nutrition.calories -= calories
	if g.nutrition.calories < 0 {
		g.nutrition.calories = 0
	}
	g.nutrition.protein -= protein
	if g.nutrition.protein < 0 {
		g.nutrition.protein = 0
	}
	g.nutrition.carbs -= carbs
	if g.nutrition.carbs < 0 {
		g.nutrition.carbs = 0
	}
	g.nutrition.fat -= fat
	if g.nutrition.fat < 0 {
		g.nutrition.fat = 0
	}
}
func (g *Game) eatBestFood() {
	if len(g.foods) == 0 {
		return
	}
	best := 0
	for i := range g.foods {
		if g.foods[i].calories > g.foods[best].calories {
			best = i
		}
	}
	meal := g.foods[best]
	g.foods = append(g.foods[:best], g.foods[best+1:]...)
	if g.rng.Float64() < meal.risk {
		g.sickTimer = 8
		g.hunger -= 8
		if g.hunger < 0 {
			g.hunger = 0
		}
		g.log("sickness", meal.name)
		g.message = fmt.Sprintf("The %s makes you sick. You misidentified a risky plant.", meal.name)
		return
	}
	restore := meal.calories / 20
	g.hunger += restore
	if g.hunger > 100 {
		g.hunger = 100
	}
	g.nutrition.calories += meal.calories
	g.nutrition.protein += meal.protein
	g.nutrition.carbs += meal.carbs
	g.nutrition.fat += meal.fat
	g.nutrition.fiber += meal.fiber
	g.log("meal", fmt.Sprintf("%s restored %d hunger", meal.name, restore))
	g.message = fmt.Sprintf("Automatic meal: %s restores %d hunger.", meal.name, restore)
}
func (g *Game) log(kind, details string) {
	g.events = append(g.events, LogEvent{Hour: float64(g.day-1)*24 + g.hour, Kind: kind, Details: details})
	if len(g.events) > 2000 {
		g.events = g.events[len(g.events)-2000:]
	}
}
func (g *Game) runHours() int { return (g.day-1)*24 + int(g.hour) }
func (g *Game) finishRun() {
	if g.dead {
		return
	}
	g.dead = true
	hours := g.runHours()
	if qualifiesScore(g.scores, hours) {
		g.scores = recordScore(g.scores, hours, getInitials())
	}
	g.log("death", fmt.Sprintf("run ended after %d hours", hours))
	saveRunLog(g.events, g.runHours())
	g.message = "You did not make it. Press Escape to try again."
}
func (g *Game) tick() {
	if g.hunger <= 35 && len(g.foods) > 0 {
		g.eatBestFood()
	}
	g.hunger--
	if g.sickTimer > 0 {
		g.sickTimer--
		g.warmth--
	}
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
		g.finishRun()
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
		if n.kind == "wood" {
			ebitenutil.DrawRect(screen, float64(n.x-8), float64(n.y-12), 16, 24, color.RGBA{38, 64, 40, 255})
			ebitenutil.DrawRect(screen, float64(n.x-13), float64(n.y-5), 26, 7, color.RGBA{102, 67, 41, 255})
		}
		if n.kind == "plant" {
			c := color.RGBA{96, 180, 78, 255}
			if n.food.risk > 0 {
				c = color.RGBA{154, 93, 167, 255}
			}
			ebitenutil.DrawRect(screen, float64(n.x-7), float64(n.y-10), 14, 20, c)
			ebitenutil.DrawRect(screen, float64(n.x-11), float64(n.y-3), 22, 6, color.RGBA{74, 135, 65, 255})
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
	text.Draw(screen, fmt.Sprintf("HUNGER %d   WARMTH %d   WOOD %d   FOOD %d", g.hunger, g.warmth, g.wood, len(g.foods)), basicfont.Face7x13, 32, 397, color.White)
	interactionText, interactionColor := g.interactionLabel()
	text.Draw(screen, "WASD / ARROWS move", basicfont.Face7x13, 32, 420, color.White)
	text.Draw(screen, interactionText, basicfont.Face7x13, 190, 420, interactionColor)
	text.Draw(screen, "ESC restart", basicfont.Face7x13, 300, 420, color.White)
	text.Draw(screen, "1 fire", basicfont.Face7x13, 32, 442, g.actionColor(g.near("camp") && g.wood >= 2))
	text.Draw(screen, "2 shelter", basicfont.Face7x13, 88, 442, g.actionColor(g.near("camp") && !g.shelter && g.wood >= 6))
	text.Draw(screen, "3 empty", basicfont.Face7x13, 160, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "4 empty", basicfont.Face7x13, 220, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "5 empty", basicfont.Face7x13, 280, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "6 empty", basicfont.Face7x13, 340, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "7 empty", basicfont.Face7x13, 400, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "8 empty", basicfont.Face7x13, 460, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "9 empty", basicfont.Face7x13, 520, 442, color.RGBA{150, 160, 155, 255})
	text.Draw(screen, "0 empty", basicfont.Face7x13, 580, 442, color.RGBA{150, 160, 155, 255})

	text.Draw(screen, g.message, basicfont.Face7x13, 32, 464, color.RGBA{240, 220, 160, 255})
	if g.dead {
		ebitengineDrawLeaderboard(screen, g)
	}
}
func ebitengineDrawLeaderboard(screen *ebiten.Image, g *Game) {
	ebitenutil.DrawRect(screen, 150, 70, 340, 285, color.RGBA{12, 18, 20, 235})
	text.Draw(screen, "RUN OVER", basicfont.Face7x13, 285, 105, color.RGBA{245, 220, 150, 255})
	text.Draw(screen, fmt.Sprintf("SURVIVED %d IN-GAME HOURS", g.runHours()), basicfont.Face7x13, 215, 130, color.White)
	text.Draw(screen, "TOP 100 LOCAL LEADERBOARD", basicfont.Face7x13, 230, 170, color.White)
	for i, score := range g.scores {
		text.Draw(screen, fmt.Sprintf("%d. %s  %d hours", i+1, score.Name, score.Hours), basicfont.Face7x13, 225, 195+i*22, color.RGBA{210, 225, 215, 255})
	}
	text.Draw(screen, "ESC to start a new run", basicfont.Face7x13, 235, 330, color.RGBA{240, 220, 160, 255})
}
func (g *Game) Layout(_, _ int) (int, int) { return 640, 480 }
func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Last Light")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}

