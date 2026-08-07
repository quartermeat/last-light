package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
)

type point struct{ x, y float64 }
type food struct{ calories, risk int }
type strategy struct {
	woodTarget int
	plants     int
	hunts      int
	fishFirst  bool
}
type run struct {
	hourTotalCount int
	hunger, warmth int
	wood, calories int
	fire, shelter  bool
	fuelHours      int
	position       point
	woods, plants  []point
	animals        []point
	foods          []food
	rng            *rand.Rand
	weather        int
}

func main() {
	trials := flag.Int("trials", 10000, "number of strategies to try")
	seed := flag.Int64("seed", 1, "random seed")
	flag.Parse()

	bestHours := -1
	best := strategy{}
	master := rand.New(rand.NewSource(*seed))
	for i := 0; i < *trials; i++ {
		s := strategy{woodTarget: 8 + master.Intn(6), plants: master.Intn(5), hunts: master.Intn(4), fishFirst: master.Intn(2) == 0}
		hours := simulate(s, master.Int63())
		if hours > bestHours {
			bestHours, best = hours, s
		}
	}

	fmt.Printf("best survival: %d in-game hours\n", bestHours)
	fmt.Printf("strategy: wood target=%d, plants=%d, hunts=%d, fish-first=%t\n", best.woodTarget, best.plants, best.hunts, best.fishFirst)
}

func simulate(s strategy, seed int64) int {
	g := &run{
		hunger: 75, warmth: 70, wood: 3, position: point{320, 250},
		rng:     rand.New(rand.NewSource(seed)),
		woods:   []point{{110, 120}, {170, 260}, {500, 150}, {535, 255}, {95, 305}, {210, 210}, {210, 285}, {315, 210}, {315, 285}, {300, 300}},
		plants:  []point{{450, 285}, {145, 185}, {330, 125}, {470, 230}},
		animals: []point{{390, 120}, {520, 300}, {210, 170}},
	}

	for g.wood < s.woodTarget && len(g.woods) > 0 && !g.dead() {
		g.move(g.woods[0])
		g.woods = g.woods[1:]
		g.wood++
		g.action(20)
	}
	if g.wood >= 8 && !g.dead() {
		g.wood -= 8
		g.shelter, g.fire = true, true
	}
	if s.fishFirst {
		g.fish()
	}
	for i := 0; i < s.plants && len(g.plants) > 0 && !g.dead(); i++ {
		g.move(g.plants[0])
		g.plants = g.plants[1:]
		g.foods = append(g.foods, food{calories: 80 + g.rng.Intn(241)})
		g.action(15)
	}
	for i := 0; i < s.hunts && len(g.animals) > 0 && !g.dead(); i++ {
		g.move(g.animals[0])
		g.animals = g.animals[1:]
		g.action(60)
		if g.rng.Float64() < .5 {
			g.foods = append(g.foods, food{calories: 600})
		}
	}
	if !s.fishFirst && !g.dead() {
		g.fish()
	}
	for !g.dead() && g.hourTotal() < 10000 {
		g.fish()
	}
	return g.hourTotal()
}

func (g *run) move(target point) {
	hours := int(math.Ceil(math.Hypot(target.x-g.position.x, target.y-g.position.y) / 70))
	if hours < 1 {
		hours = 1
	}
	for i := 0; i < hours && !g.dead(); i++ {
		g.action(35)
		g.tick()
	}
	g.position = target
}

func (g *run) fish() {
	if g.dead() {
		return
	}
	g.move(point{320, 70})
	g.action(35)
	if g.rng.Float64() < .55 {
		g.foods = append(g.foods, food{calories: 500})
	}
	g.tick()
}

func (g *run) action(calories int) {
	if g.calories < calories {
		g.hunger--
	}
	g.calories -= calories
	if g.calories < 0 {
		g.calories = 0
	}
}

func (g *run) tick() {
	if g.hunger <= 35 && len(g.foods) > 0 {
		best := 0
		for i := range g.foods {
			if g.foods[i].calories > g.foods[best].calories {
				best = i
			}
		}
		meal := g.foods[best]
		g.foods = append(g.foods[:best], g.foods[best+1:]...)
		if g.rng.Intn(100) >= meal.risk {
			g.hunger += meal.calories / 20
			if g.hunger > 100 {
				g.hunger = 100
			}
			g.calories += meal.calories
		} else {
			g.hunger -= 8
		}
	}
	g.hunger--
	if g.fire {
		g.fuelHours++
		if g.fuelHours >= 4 {
			g.fuelHours = 0
			if g.wood > 0 {
				g.wood--
			} else {
				g.fire = false
			}
		}
		if g.fire {
			g.warmth += 2
		}
	} else {
		g.warmth--
	}
	if g.rng.Intn(3) == 1 {
		g.weather = 1
	}
	if g.weather == 1 && !g.shelter {
		g.warmth -= 2
	}
	g.hourTotalCount++
}

func (g *run) dead() bool     { return g.hunger <= 0 || g.warmth <= 0 }
func (g *run) hourTotal() int { return g.hourTotalCount }
