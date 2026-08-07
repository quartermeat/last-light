package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

const tile = 32
const mapCols = 18
const mapRows = 10
const waterTileCount = 18
const maxWood = 20
const version = "v0.13.2"

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
type ReplayFrame struct {
	Hour     float64 `json:"hour"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Hunger   int     `json:"hunger"`
	Warmth   int     `json:"warmth"`
	Wood     int     `json:"wood"`
	Fire     bool    `json:"fire"`
	Shelter  bool    `json:"shelter"`
	FireX    float64 `json:"fire_x"`
	FireY    float64 `json:"fire_y"`
	ShelterX float64 `json:"shelter_x"`
	ShelterY float64 `json:"shelter_y"`
}
type Game struct {
	day                                       int
	hour, tickTimer, moveTimer, fireBurnHours float64
	hunger, warmth, wood                      int
	shelter, fire                             bool
	fireX, fireY, shelterX, shelterY          float64
	weather, message                          string
	x, y                                      float64
	nodes                                     []Node
	animals                                   []Animal
	foods                                     []Food
	rng                                       *rand.Rand
	nutrition                                 Nutrition
	sickTimer                                 int
	quarterTicks                              int
	scores                                    []Score
	submittedScore                            Score
	submittedRank                             int
	dead                                      bool
	events                                    []LogEvent
	replayFrames                              []ReplayFrame
	replaying                                 bool
	replayIndex                               int
	water                                     [mapCols][mapRows]bool
}

func NewGame() *Game {
	berries := Food{"shore berries", 120, 1, 28, 0, 4, 0}
	g := &Game{day: 1, warmth: 70, hunger: 75, wood: 3, weather: "clear", x: 320, y: 250, rng: rand.New(rand.NewSource(time.Now().UnixNano())), message: "Explore the island. Q interacts with whatever is closest.", foods: []Food{berries, berries}, nodes: []Node{
		{110, 120, "wood", false, Food{}}, {170, 260, "wood", false, Food{}}, {500, 150, "wood", false, Food{}}, {535, 255, "wood", false, Food{}}, {95, 305, "wood", false, Food{}}, {210, 210, "wood", false, Food{}}, {210, 285, "wood", false, Food{}}, {315, 210, "wood", false, Food{}}, {315, 285, "wood", false, Food{}}, {300, 300, "wood", false, Food{}},
		{450, 285, "plant", false, Food{"shore berries", 120, 1, 28, 0, 4, 0}}, {145, 185, "plant", false, Food{"wild greens", 80, 3, 10, 0, 5, 0}}, {330, 125, "plant", false, Food{"edible root", 320, 4, 72, 1, 8, 0}}, {470, 230, "plant", false, Food{"questionable mushroom", 60, 2, 8, 1, 2, .35}}, {260, 245, "camp", false, Food{}},
	}, animals: []Animal{{390, 120, .5, .2, true, 1}, {520, 300, -.3, .4, true, 2}, {210, 170, .2, -.4, true, 3}}, scores: loadScores()}
	g.generateTerrain()
	startLeaderboardSync(g)
	return g
}

func (g *Game) generateTerrain() {
	protected := make(map[[2]int]bool)
	protect := func(x, y float64) {
		col, row, ok := g.cellForPixel(x, y)
		if ok {
			protected[[2]int{col, row}] = true
		}
	}
	protect(g.x, g.y)
	for _, n := range g.nodes {
		protect(float64(n.x), float64(n.y))
	}
	for _, a := range g.animals {
		protect(a.x, a.y)
	}
	placed := 0
	for _, index := range g.rng.Perm(mapCols * mapRows) {
		if placed >= waterTileCount {
			break
		}
		cell := [2]int{index % mapCols, index / mapCols}
		if protected[cell] {
			continue
		}
		g.water[cell[0]][cell[1]] = true
		placed++
	}
}

func (g *Game) cellForPixel(x, y float64) (int, int, bool) {
	col := int((x - 32) / tile)
	row := int((y - 48) / tile)
	if col < 0 || col >= mapCols || row < 0 || row >= mapRows {
		return 0, 0, false
	}
	return col, row, true
}

func (g *Game) isWaterAt(x, y float64) bool {
	col, row, ok := g.cellForPixel(x, y)
	return ok && g.water[col][row]
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		*g = *NewGame()
		return nil
	}
	if g.replaying {
		g.advanceReplay()
		return nil
	}
	if g.dead {
		g.selectReplay()
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
	nextX, nextY := g.x+dx*2.2, g.y+dy*2.2
	if !g.isWaterAt(nextX, nextY) {
		g.x, g.y = nextX, nextY
	}
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
	if inpututil.IsKeyJustPressed(ebiten.Key1) && !g.fire && g.wood >= 2 {
		g.expend(30, 0, 5, 1)
		g.wood -= 2
		g.fire = true
		g.fireX, g.fireY = g.x, g.y
		g.log("fire", "lit fire at current location")
		g.message = "The fire catches. Warmth returns."
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) && !g.shelter && g.wood >= 6 {
		g.expend(60, 0, 8, 2)
		g.wood -= 6
		g.shelter = true
		g.shelterX, g.shelterY = g.x, g.y
		g.log("shelter", "built shelter at current location")
		g.message = "A rough shelter stands against the wind."
	}
	g.hour += 1.0 / 60.0
	if g.hour >= 24 {
		g.day++
		g.hour = 0
		g.weather = []string{"clear", "wind", "rain"}[g.rng.Intn(3)]
	}
	g.tickTimer += 1.0 / 60.0
	if g.tickTimer >= 0.25 {
		g.tickTimer -= 0.25
		g.tick()
	}
	return nil
}

func (g *Game) selectReplay() {
	keys := []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6}
	for i, key := range keys {
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		var score Score
		if g.submittedRank > 6 && i == 5 {
			score = g.submittedScore
		} else if i < len(g.scores) && i < 6 {
			score = g.scores[i]
		} else {
			return
		}
		if len(score.Replay) == 0 {
			g.message = "That run predates replay capture. New runs can be replayed."
			return
		}
		g.replaying = true
		g.replayIndex = 0
		g.message = fmt.Sprintf("Replaying %s at 8x speed. ESC to stop.", score.Name)
		g.applyReplayFrame(score.Replay[0])
		g.replayFrames = score.Replay
		return
	}
}

func (g *Game) advanceReplay() {
	if g.replayIndex >= len(g.replayFrames) {
		g.replaying = false
		g.dead = true
		g.message = "Replay finished. Press 1-6 to replay another run, or ESC to restart."
		return
	}
	end := g.replayIndex + 8
	if end > len(g.replayFrames) {
		end = len(g.replayFrames)
	}
	g.applyReplayFrame(g.replayFrames[end-1])
	g.replayIndex = end
}

func (g *Game) applyReplayFrame(frame ReplayFrame) {
	g.day = int(frame.Hour/24) + 1
	g.hour = frame.Hour - float64((g.day-1)*24)
	g.x, g.y = frame.X, frame.Y
	g.hunger, g.warmth, g.wood = frame.Hunger, frame.Warmth, frame.Wood
	g.fire, g.shelter = frame.Fire, frame.Shelter
	g.fireX, g.fireY = frame.FireX, frame.FireY
	g.shelterX, g.shelterY = frame.ShelterX, frame.ShelterY
}
func (g *Game) updateAnimals() {
	for i := range g.animals {
		a := &g.animals[i]
		if !a.active {
			continue
		}
		nextX, nextY := a.x+a.vx, a.y+a.vy
		if g.isWaterAt(nextX, nextY) {
			a.vx *= -1
			a.vy *= -1
		} else {
			a.x, a.y = nextX, nextY
		}
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
	best, bestDistance := -1, 999999.0
	for i := range g.nodes {
		n := &g.nodes[i]
		if n.used || abs(float64(n.x)-g.x) >= 42 || abs(float64(n.y)-g.y) >= 42 {
			continue
		}
		if n.kind != "plant" && !(n.kind == "wood" && g.wood < maxWood) {
			continue
		}
		dx, dy := float64(n.x)-g.x, float64(n.y)-g.y
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			best, bestDistance = i, distance
		}
	}
	if best < 0 {
		g.message = "Nothing useful is close enough to gather."
		return
	}
	n := &g.nodes[best]
	if n.kind == "wood" {
		g.expend(20, 1, 3, 0)
		g.wood++
		n.used = true
		g.log("gather", "gathered wood")
		g.message = "You gather dry wood. No luck involved."
		return
	}
	g.expend(15, 0, 2, 0)
	g.foods = append(g.foods, n.food)
	n.used = true
	g.log("gather", n.food.name)
	g.message = fmt.Sprintf("You gather %s. It will be eaten automatically when hunger is low.", n.food.name)
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
func (g *Game) nearWater() bool {
	col, row, ok := g.cellForPixel(g.x, g.y)
	if !ok {
		return false
	}
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			checkCol, checkRow := col+dx, row+dy
			if checkCol >= 0 && checkCol < mapCols && checkRow >= 0 && checkRow < mapRows && g.water[checkCol][checkRow] {
				return true
			}
		}
	}
	return false
}
func (g *Game) canGather() bool {
	for _, n := range g.nodes {
		if n.used || abs(float64(n.x)-g.x) >= 42 || abs(float64(n.y)-g.y) >= 42 {
			continue
		}
		if n.kind == "plant" || (n.kind == "wood" && g.wood < maxWood) {
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

func (g *Game) sunPosition() (float64, bool) {
	if g.hour < 6 || g.hour >= 18 {
		return 0, false
	}
	return ((g.hour - 6) / 12) * 17, true
}

func (g *Game) sunHeatAtPlayer() int {
	heat := g.sunHeatAtPlayerFloat()
	if heat <= 0 {
		return 0
	}
	return int(math.Round(heat))
}

func (g *Game) sunHeatAtPlayerFloat() float64 {
	sun, visible := g.sunPosition()
	if !visible {
		return 0
	}
	playerColumn := int((g.x - 32) / tile)
	distance := sun - (float64(playerColumn) + 0.5)
	if distance < 0 {
		distance = -distance
	}
	heat := 5 - distance
	if heat < 0 {
		return 0
	}
	return heat
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
	name := "RUN"
	if qualifiesScore(g.scores, hours) {
		name = getInitials()
		g.submittedScore = Score{Name: name, Hours: hours, Replay: append([]ReplayFrame(nil), g.replayFrames...)}
		g.scores = recordScore(g.scores, hours, name)
		for i := range g.scores {
			if g.scores[i].Name == g.submittedScore.Name && g.scores[i].Hours == g.submittedScore.Hours {
				g.scores[i].Replay = g.submittedScore.Replay
				break
			}
		}
		saveScores(g.scores)
		g.submittedRank = scoreRank(g.scores, g.submittedScore)
	}
	g.log("death", fmt.Sprintf("run ended after %d hours", hours))
	saveRunLog(g.events, hours, name, g.replayFrames)
	g.message = "You did not make it. Press Escape to try again."
}
func (g *Game) tick() {
	g.quarterTicks++
	if g.quarterTicks >= 4 {
		g.quarterTicks = 0
		if g.hunger <= 35 && len(g.foods) > 0 {
			g.eatBestFood()
		}
		g.hunger--
		if g.sickTimer > 0 {
			g.sickTimer--
			g.warmth--
		}
	}
	if g.fire {
		g.fireBurnHours += 0.25
		if g.fireBurnHours >= 4 {
			if g.wood > 0 {
				g.wood--
				g.fireBurnHours = 0
				g.log("fire_fuel", "fire consumed 1 wood")
			} else {
				g.fire = false
				g.log("fire_out", "fire ran out of wood")
				g.message = "The fire goes out. There is no wood left to feed it."
			}
		}
		if g.fire && g.quarterTicks == 0 {
			g.warmth += 2
		}
	} else if g.quarterTicks == 0 {
		g.warmth--
	}
	if g.quarterTicks == 0 {
		if g.weather == "wind" {
			g.warmth--
		}
		if g.weather == "rain" && !g.shelter {
			g.warmth -= 2
		}
		g.warmth += g.sunHeatAtPlayer()
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
	g.recordReplayFrame()
}

func (g *Game) recordReplayFrame() {
	if len(g.replayFrames) >= 4000 {
		return
	}
	g.replayFrames = append(g.replayFrames, ReplayFrame{
		Hour: g.runHoursFloat(), X: g.x, Y: g.y,
		Hunger: g.hunger, Warmth: g.warmth, Wood: g.wood,
		Fire: g.fire, Shelter: g.shelter,
		FireX: g.fireX, FireY: g.fireY,
		ShelterX: g.shelterX, ShelterY: g.shelterY,
	})
}

func (g *Game) runHoursFloat() float64 { return float64(g.day-1)*24 + g.hour }

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{22, 31, 35, 255})
	ebitenutil.DrawRect(screen, 32, 48, 576, 320, color.RGBA{53, 91, 72, 255})
	for row := 0; row < mapRows; row++ {
		for col := 0; col < mapCols; col++ {
			x, y := 32+col*tile, 48+row*tile
			brightness := g.sunBrightness(col)
			if g.water[col][row] {
				ebitenutil.DrawRect(screen, float64(x), float64(y), tile, tile, scaleColor(color.RGBA{52, 112, 143, 255}, brightness))
				ebitenutil.DrawRect(screen, float64(x+3), float64(y+9), tile-6, 2, scaleColor(color.RGBA{125, 190, 205, 180}, brightness))
				continue
			}
			c := color.RGBA{58, 98, 73, 255}
			if (x/tile+y/tile)%3 == 0 {
				c = color.RGBA{61, 103, 76, 255}
			}
			ebitenutil.DrawRect(screen, float64(x), float64(y), tile-1, tile-1, scaleColor(c, brightness))
		}
	}
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
		ebitenutil.DrawRect(screen, g.shelterX-31, g.shelterY-19, 62, 38, color.RGBA{91, 57, 43, 255})
	}
	if g.fire {
		ebitenutil.DrawRect(screen, g.fireX-14, g.fireY-7, 28, 14, color.RGBA{235, 126, 43, 255})
	}
	ebitenutil.DrawRect(screen, g.x-8, g.y-8, 16, 16, color.RGBA{231, 194, 142, 255})
	ebitenutil.DrawRect(screen, g.x-5, g.y-13, 10, 7, color.RGBA{85, 52, 36, 255})
	g.drawFogOverlay(screen)
	minutes := int((g.hour - float64(int(g.hour))) * 60)
	minutes = (minutes / 15) * 15
	text.Draw(screen, fmt.Sprintf("LAST LIGHT   DAY %d   %02d:%02d   %s", g.day, int(g.hour), minutes, g.weather), basicfont.Face7x13, 32, 28, color.White)
	text.Draw(screen, version, basicfont.Face7x13, 575, 28, color.RGBA{180, 205, 190, 255})
	rainStatus, rainColor := "NO RAIN PENALTY", color.RGBA{180, 205, 190, 255}
	if g.weather == "rain" && g.shelter {
		rainStatus, rainColor = "RAIN: SHELTERED", color.RGBA{120, 235, 145, 255}
	} else if g.weather == "rain" {
		rainStatus, rainColor = "RAIN: -2 WARMTH", color.RGBA{245, 110, 100, 255}
	}
	text.Draw(screen, rainStatus, basicfont.Face7x13, 32, 44, rainColor)
	if g.hour < 6 || g.hour >= 18 {
		text.Draw(screen, "NIGHT: LIMITED VISIBILITY", basicfont.Face7x13, 230, 44, color.RGBA{230, 190, 120, 255})
	}
	text.Draw(screen, fmt.Sprintf("HUNGER %d   WARMTH %d   WOOD %d   FOOD %d", g.hunger, g.warmth, g.wood, len(g.foods)), basicfont.Face7x13, 32, 397, color.White)
	interactionText, interactionColor := g.interactionLabel()
	text.Draw(screen, "WASD / ARROWS move", basicfont.Face7x13, 32, 420, color.White)
	text.Draw(screen, interactionText, basicfont.Face7x13, 190, 420, interactionColor)
	text.Draw(screen, "ESC restart", basicfont.Face7x13, 300, 420, color.White)
	text.Draw(screen, "1 fire", basicfont.Face7x13, 32, 442, g.actionColor(!g.fire && g.wood >= 2))
	text.Draw(screen, "2 shelter", basicfont.Face7x13, 88, 442, g.actionColor(!g.shelter && g.wood >= 6))
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

func (g *Game) sunBrightness(column int) float64 {
	sun, visible := g.sunPosition()
	if !visible {
		return 1
	}
	distance := sun - (float64(column) + 0.5)
	if distance < 0 {
		distance = -distance
	}
	if distance > 9 {
		distance = 9
	}
	return 1.08 - distance*0.025
}

func scaleColor(c color.RGBA, brightness float64) color.RGBA {
	channel := func(value uint8) uint8 {
		scaled := int(float64(value) * brightness)
		if scaled < 0 {
			return 0
		}
		if scaled > 255 {
			return 255
		}
		return uint8(scaled)
	}
	return color.RGBA{channel(c.R), channel(c.G), channel(c.B), c.A}
}

func (g *Game) drawFogOverlay(screen *ebiten.Image) {
	const mapLeft, mapTop, mapRight, mapBottom = 32.0, 48.0, 608.0, 368.0
	sunHeat := g.sunHeatAtPlayerFloat()
	if sunHeat >= 4.5 {
		return
	}
	radius := 54.0 + sunHeat*70
	alpha := uint8(225)
	if g.hour >= 6 && g.hour < 18 {
		alpha = 205
	}
	for y := mapTop; y < mapBottom; y += 4 {
		intervals := make([][2]float64, 0, 2)
		addLight := func(centerX, centerY, lightRadius float64) {
			distanceY := (y + 2) - centerY
			if distanceY < 0 {
				distanceY = -distanceY
			}
			if distanceY >= lightRadius {
				return
			}
			halfWidth := math.Sqrt(lightRadius*lightRadius - distanceY*distanceY)
			intervals = append(intervals, [2]float64{centerX - halfWidth, centerX + halfWidth})
		}
		addLight(g.x, g.y, radius)
		if g.fire {
			addLight(g.fireX, g.fireY, 115)
		}
		if len(intervals) == 2 && intervals[1][0] < intervals[0][0] {
			intervals[0], intervals[1] = intervals[1], intervals[0]
		}
		if len(intervals) == 2 && intervals[1][0] <= intervals[0][1] {
			if intervals[1][1] > intervals[0][1] {
				intervals[0][1] = intervals[1][1]
			}
			intervals = intervals[:1]
		}
		cursor := mapLeft
		for _, interval := range intervals {
			left, right := interval[0], interval[1]
			if left < mapLeft {
				left = mapLeft
			}
			if right > mapRight {
				right = mapRight
			}
			if left > cursor {
				ebitenutil.DrawRect(screen, cursor, y, left-cursor, 4, color.RGBA{5, 9, 16, alpha})
			}
			if right > cursor {
				cursor = right
			}
		}
		if cursor < mapRight {
			ebitenutil.DrawRect(screen, cursor, y, mapRight-cursor, 4, color.RGBA{5, 9, 16, alpha})
		}
	}
}

func ebitengineDrawLeaderboard(screen *ebiten.Image, g *Game) {
	ebitenutil.DrawRect(screen, 150, 70, 340, 285, color.RGBA{12, 18, 20, 235})
	text.Draw(screen, "RUN OVER", basicfont.Face7x13, 285, 105, color.RGBA{245, 220, 150, 255})
	text.Draw(screen, fmt.Sprintf("SURVIVED %d IN-GAME HOURS", g.runHours()), basicfont.Face7x13, 215, 130, color.White)
	text.Draw(screen, "TOP 6 / 100 LOCAL LEADERBOARD", basicfont.Face7x13, 215, 170, color.White)
	visibleScores := len(g.scores)
	if g.submittedRank > 6 {
		visibleScores = 5
	} else if visibleScores > 6 {
		visibleScores = 6
	}
	for i := 0; i < visibleScores; i++ {
		score := g.scores[i]
		rowColor := color.RGBA{210, 225, 215, 255}
		if g.submittedRank > 0 && score.Name == g.submittedScore.Name && score.Hours == g.submittedScore.Hours {
			rowColor = color.RGBA{255, 220, 100, 255}
		}
		text.Draw(screen, fmt.Sprintf("[%d] %d. %s  %d hours", i+1, i+1, score.Name, score.Hours), basicfont.Face7x13, 220, 195+i*22, rowColor)
	}
	if g.submittedRank > 6 {
		text.Draw(screen, fmt.Sprintf("[6] %d. %s  %d hours  (YOUR RUN)", g.submittedRank, g.submittedScore.Name, g.submittedScore.Hours), basicfont.Face7x13, 220, 195+5*22, color.RGBA{255, 220, 100, 255})
	}
	text.Draw(screen, "1-6: replay visible run   ESC: new run", basicfont.Face7x13, 205, 330, color.RGBA{240, 220, 160, 255})
}
func (g *Game) Layout(_, _ int) (int, int) { return 640, 480 }
func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Last Light")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}
