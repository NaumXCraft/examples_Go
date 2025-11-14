// file: dungeon.go
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
Simple ASCII Dungeon Crawler.

Run: go run dungeon.go
Controls:
 w/a/s/d - move
 i       - show inventory
 p       - pick up item on current tile
 q       - quit
*/

type TileType int

const (
	EmptyTile TileType = iota
	WallTile
	FloorTile
)

type Tile struct {
	X       int
	Y       int
	Type    TileType
	Item    *Item
	Entity  *Entity
	Visible bool
}

type Item struct {
	ID   string
	Name string
	// consumable: heal
	Heal int
}

type Stats struct {
	HPMax   int
	HP      int
	Attack  int
	Defense int
	Speed   int
}

type Entity struct {
	ID       string
	Name     string
	X, Y     int
	Stats    Stats
	Inv      []Item
	IsPlayer bool
	Alive    bool
	AIType   string // "basic" chase
}

type World struct {
	Width    int
	Height   int
	Tiles    [][]*Tile
	Player   *Entity
	Entities []*Entity
	Rand     *rand.Rand
}

func NewWorld(w, h int, r *rand.Rand) *World {
	tiles := make([][]*Tile, h)
	for y := 0; y < h; y++ {
		row := make([]*Tile, w)
		for x := 0; x < w; x++ {
			row[x] = &Tile{X: x, Y: y, Type: FloorTile}
		}
		tiles[y] = row
	}
	world := &World{Width: w, Height: h, Tiles: tiles, Rand: r}
	world.makeWalls()
	return world
}

func (world *World) makeWalls() {
	// border walls
	for x := 0; x < world.Width; x++ {
		world.Tiles[0][x].Type = WallTile
		world.Tiles[world.Height-1][x].Type = WallTile
	}
	for y := 0; y < world.Height; y++ {
		world.Tiles[y][0].Type = WallTile
		world.Tiles[y][world.Width-1].Type = WallTile
	}
	// random interior walls
	for i := 0; i < (world.Width*world.Height)/10; i++ {
		x := world.Rand.Intn(world.Width-2) + 1
		y := world.Rand.Intn(world.Height-2) + 1
		world.Tiles[y][x].Type = WallTile
	}
}

func (w *World) PlaceEntity(e *Entity, x, y int) {
	e.X = x
	e.Y = y
	e.Alive = true
	w.Tiles[y][x].Entity = e
	w.Entities = append(w.Entities, e)
	if e.IsPlayer {
		w.Player = e
	}
}

func (w *World) PlaceItem(it Item, x, y int) {
	w.Tiles[y][x].Item = &it
}

// move entity if possible
func (w *World) MoveEntity(e *Entity, nx, ny int) bool {
	if nx < 0 || nx >= w.Width || ny < 0 || ny >= w.Height {
		return false
	}
	dest := w.Tiles[ny][nx]
	if dest.Type == WallTile {
		return false
	}
	if dest.Entity != nil {
		// attack if hostile
		if e.IsPlayer != dest.Entity.IsPlayer {
			w.ResolveMelee(e, dest.Entity)
			return true
		}
		return false
	}
	// move
	w.Tiles[e.Y][e.X].Entity = nil
	e.X = nx
	e.Y = ny
	dest.Entity = e
	return true
}

func (w *World) ResolveMelee(attacker, defender *Entity) {
	// very simple damage calc
	damage := attacker.Stats.Attack - defender.Stats.Defense/2
	if damage < 1 {
		damage = 1
	}
	fmt.Printf("%s атакует %s на %d урона\n", attacker.Name, defender.Name, damage)
	defender.Stats.HP -= damage
	if defender.Stats.HP <= 0 {
		defender.Stats.HP = 0
		defender.Alive = false
		fmt.Printf("%s убит(а)!\n", defender.Name)
		// remove from tile
		w.Tiles[defender.Y][defender.X].Entity = nil
	}
}

func (w *World) RemoveDeadEntities() {
	newlist := make([]*Entity, 0, len(w.Entities))
	for _, e := range w.Entities {
		if e.Alive {
			newlist = append(newlist, e)
		}
	}
	w.Entities = newlist
}

// simple BFS to find direction to player (returns dx,dy of next step) or 0,0 if none
func (w *World) BFSStepTowards(src *Entity, target *Entity) (int, int) {
	type node struct{ x, y int }
	H := w.Height
	W := w.Width
	vis := make([][]bool, H)
	for i := 0; i < H; i++ {
		vis[i] = make([]bool, W)
	}
	prev := make(map[node]node)
	q := []node{{src.X, src.Y}}
	vis[src.Y][src.X] = true
	found := false
	var dest node
	dirs := []node{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(q) > 0 && !found {
		cur := q[0]
		q = q[1:]
		for _, d := range dirs {
			nx := cur.x + d.x
			ny := cur.y + d.y
			if nx < 0 || nx >= W || ny < 0 || ny >= H {
				continue
			}
			if vis[ny][nx] {
				continue
			}
			t := w.Tiles[ny][nx]
			if t.Type == WallTile {
				continue
			}
			vis[ny][nx] = true
			prev[node{nx, ny}] = cur
			if nx == target.X && ny == target.Y {
				found = true
				dest = node{nx, ny}
				break
			}
			// can step through entities for pathfinding (avoid if entity is wall-like)
			q = append(q, node{nx, ny})
		}
	}
	if !found {
		return 0, 0
	}
	// walk back to neighbor of src
	cur := dest
	for {
		p := prev[cur]
		if p.x == src.X && p.y == src.Y {
			// cur is the next cell to step into
			return cur.x - src.X, cur.y - src.Y
		}
		cur = p
	}
}

// render map ASCII
func (w *World) Render() {
	fmt.Println()
	for y := 0; y < w.Height; y++ {
		var line strings.Builder
		for x := 0; x < w.Width; x++ {
			t := w.Tiles[y][x]
			ch := '.'
			if t.Type == WallTile {
				ch = '#'
			} else if t.Entity != nil {
				if t.Entity.IsPlayer {
					ch = '@'
				} else {
					ch = 'g' // generic monster
				}
			} else if t.Item != nil {
				ch = '!'
			}
			line.WriteRune(rune(ch))
		}
		fmt.Println(line.String())
	}
	fmt.Printf("HP: %d/%d\n", w.Player.Stats.HP, w.Player.Stats.HPMax)
}

func (w *World) PlayerPickUp() {
	t := w.Tiles[w.Player.Y][w.Player.X]
	if t.Item == nil {
		fmt.Println("Здесь нет предметов")
		return
	}
	it := *t.Item
	w.Player.Inv = append(w.Player.Inv, it)
	fmt.Printf("Подобрали: %s\n", it.Name)
	t.Item = nil
}

func (w *World) PlayerUseItem(idx int) {
	if idx < 0 || idx >= len(w.Player.Inv) {
		fmt.Println("invalid index")
		return
	}
	it := w.Player.Inv[idx]
	if it.Heal > 0 {
		w.Player.Stats.HP += it.Heal
		if w.Player.Stats.HP > w.Player.Stats.HPMax {
			w.Player.Stats.HP = w.Player.Stats.HPMax
		}
		fmt.Printf("Использовано %s, восстановлено %d HP\n", it.Name, it.Heal)
	}
	// remove used
	w.Player.Inv = append(w.Player.Inv[:idx], w.Player.Inv[idx+1:]...)
}

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	w := NewWorld(25, 12, r)

	// create player
	player := &Entity{
		ID:       "player1",
		Name:     "Игрок",
		Stats:    Stats{HPMax: 30, HP: 30, Attack: 5, Defense: 2, Speed: 5},
		IsPlayer: true,
	}
	// spawn player in center
	w.PlaceEntity(player, w.Width/2, w.Height/2)

	// spawn some monsters
	for i := 0; i < 6; i++ {
		x := r.Intn(w.Width-4) + 2
		y := r.Intn(w.Height-4) + 2
		if w.Tiles[y][x].Entity != nil || w.Tiles[y][x].Type == WallTile {
			continue
		}
		mon := &Entity{
			ID:       fmt.Sprintf("m%d", i+1),
			Name:     "Гоблин",
			Stats:    Stats{HPMax: 8, HP: 8, Attack: 3, Defense: 0, Speed: 3},
			IsPlayer: false,
			AIType:   "basic",
		}
		w.PlaceEntity(mon, x, y)
	}

	// place some items
	for i := 0; i < 5; i++ {
		x := r.Intn(w.Width-4) + 2
		y := r.Intn(w.Height-4) + 2
		if w.Tiles[y][x].Item != nil || w.Tiles[y][x].Entity != nil || w.Tiles[y][x].Type == WallTile {
			continue
		}
		it := Item{ID: fmt.Sprintf("it%d", i), Name: "Фляга здоровья", Heal: 8}
		w.PlaceItem(it, x, y)
	}

	// game loop
	reader := bufio.NewReader(os.Stdin)
	for {
		// render view
		w.Render()
		if w.Player.Stats.HP <= 0 {
			fmt.Println("Вы погибли. Игра окончена.")
			return
		}
		fmt.Print("<<Command (w/a/s/d, p pick up, i inv, u use <i>, q quit)>>: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, " ")
		cmd := parts[0]
		switch cmd {
		case "q":
			fmt.Println("Выход")
			return
		case "w", "a", "s", "d":
			dx, dy := 0, 0
			if cmd == "w" {
				dy = -1
			}
			if cmd == "s" {
				dy = 1
			}
			if cmd == "a" {
				dx = -1
			}
			if cmd == "d" {
				dx = 1
			}
			w.MoveEntity(w.Player, w.Player.X+dx, w.Player.Y+dy)
		case "p":
			w.PlayerPickUp()
		case "i":
			fmt.Println("Инвентарь:")
			for i, it := range w.Player.Inv {
				fmt.Printf("[%d] %s (heal:%d)\n", i, it.Name, it.Heal)
			}
		case "u":
			if len(parts) < 2 {
				fmt.Println("u <index>")
				continue
			}
			idx, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("invalid index")
				continue
			}
			w.PlayerUseItem(idx)
		default:
			fmt.Println("Неизвестная команда")
		}

		// monsters act
		for _, e := range w.Entities {
			if e.IsPlayer || !e.Alive {
				continue
			}
			// simple chase AI: if adjacent -> attack, else step towards
			dx := w.Player.X - e.X
			dy := w.Player.Y - e.Y
			if abs(dx)+abs(dy) == 1 {
				w.ResolveMelee(e, w.Player)
			} else {
				stx, sty := w.BFSStepTowards(e, w.Player)
				if stx != 0 || sty != 0 {
					w.MoveEntity(e, e.X+stx, e.Y+sty)
				}
			}
		}
		// cleanup
		w.RemoveDeadEntities()
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
