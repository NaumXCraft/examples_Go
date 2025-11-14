package main

import (
	"fmt"
	"math/rand"
	"time"
)

// RPG — turn-based battle simulator.
//Это компактный, но насыщенный пример со сложными структурами (много полей), методами, эффектами, оружием/брони, инвентарём и простым AI.

/*
RPG Turn-based battle simulator
- Большие структуры: Character, Stats, Inventory, Equipment, Skill, Effect
- Показывает: методы с pointer receiver, вложенные структуры, срезы, карты, интерфейсы простые
- Запуск: go run main.go
*/

type DamageType string

const (
	Physical DamageType = "physical"
	Magic    DamageType = "magic"
	Pure     DamageType = "pure"
)

type Stats struct {
	HPMax    int
	HP       int
	MPMax    int
	MP       int
	Attack   int
	Defense  int
	Magic    int
	Resist   int
	Speed    int
	CritRate float64 // 0..1
	CritMult float64 // e.g. 1.5
}

func (s *Stats) Clone() Stats {
	return Stats{
		HPMax:    s.HPMax,
		HP:       s.HP,
		MPMax:    s.MPMax,
		MP:       s.MP,
		Attack:   s.Attack,
		Defense:  s.Defense,
		Magic:    s.Magic,
		Resist:   s.Resist,
		Speed:    s.Speed,
		CritRate: s.CritRate,
		CritMult: s.CritMult,
	}
}

type Item struct {
	ID          string
	Name        string
	Description string
	// for simplicity we allow items to be consumables or equippable weapon/armor via flags
	Consumable bool
	HealHP     int
	HealMP     int
	EquipSlot  string // "weapon", "armor", "" for none
	Weapon     *Weapon
	Armor      *Armor
}

type Weapon struct {
	Name       string
	DamageMin  int
	DamageMax  int
	DamageType DamageType
	// optional stat modifiers
	AttackBonus int
	MagicBonus  int
}

type Armor struct {
	Name         string
	DefenseBonus int
	ResistBonus  int
	HPBonus      int
}

type Skill struct {
	ID          string
	Name        string
	Description string
	MPCost      int
	// simple effect: damage multiplier or heal
	DamageMultiplier float64 // multiplies caster.Attack or caster.Magic depending on DamageType
	DamageType       DamageType
	HealHP           int
	TargetAll        bool // hits all enemies
}

type Effect struct {
	ID       string
	Name     string
	Duration int // turns remaining
	AtkMod   int // flat changes
	DefMod   int
	SpeedMod int
	DotHP    int // damage per turn
	From     string
}

func (e *Effect) Tick() {
	e.Duration--
}

type Inventory struct {
	Items []Item
}

func (inv *Inventory) Add(item Item) {
	inv.Items = append(inv.Items, item)
}

func (inv *Inventory) RemoveAt(i int) {
	if i < 0 || i >= len(inv.Items) {
		return
	}
	inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
}

type Character struct {
	ID      string
	Name    string
	Stats   Stats
	Weapon  *Weapon
	Armor   *Armor
	Inv     Inventory
	Skills  []Skill
	Effects []Effect
	// misc
	Alive bool
	Team  string // "player" or "enemy"
}

func NewCharacter(id, name string, baseStats Stats) *Character {
	s := baseStats.Clone()
	s.HP = s.HPMax
	s.MP = s.MPMax
	return &Character{
		ID:      id,
		Name:    name,
		Stats:   s,
		Weapon:  nil,
		Armor:   nil,
		Inv:     Inventory{},
		Skills:  []Skill{},
		Effects: []Effect{},
		Alive:   true,
	}
}

func (c *Character) EffectiveAttack() int {
	atk := c.Stats.Attack
	if c.Weapon != nil {
		atk += c.Weapon.AttackBonus
	}
	// Effects can modify attack
	for _, e := range c.Effects {
		atk += e.AtkMod
	}
	return atk
}

func (c *Character) EffectiveDefense() int {
	def := c.Stats.Defense
	if c.Armor != nil {
		def += c.Armor.DefenseBonus
	}
	for _, e := range c.Effects {
		def += e.DefMod
	}
	return def
}

func (c *Character) EffectiveSpeed() int {
	sp := c.Stats.Speed
	for _, e := range c.Effects {
		sp += e.SpeedMod
	}
	return sp
}

func (c *Character) ApplyEffectsStartTurn() {
	// apply DOTs etc before actions
	totalDot := 0
	for _, e := range c.Effects {
		if e.DotHP != 0 {
			totalDot += e.DotHP
		}
	}
	if totalDot != 0 {
		c.TakeDamage(totalDot, Pure) // dot ignores defense for simplicity
		fmt.Printf("%s получает %d урона по эффектам (DOT)\n", c.Name, totalDot)
	}
}

func (c *Character) ApplyEffectsEndTurn() {
	// decrease durations and remove expired
	new := make([]Effect, 0, len(c.Effects))
	for i := range c.Effects {
		c.Effects[i].Tick()
		if c.Effects[i].Duration > 0 {
			new = append(new, c.Effects[i])
		} else {
			fmt.Printf("Эффект %s на %s завершился\n", c.Effects[i].Name, c.Name)
		}
	}
	c.Effects = new
}

func (c *Character) TakeDamage(amount int, dtype DamageType) {
	// simple calculation: subtract defense/resist unless Pure
	actual := amount
	if dtype == Physical {
		def := c.EffectiveDefense()
		actual = amount - def/2
		if actual < 0 {
			actual = 0
		}
	} else if dtype == Magic {
		res := c.Stats.Resist
		actual = amount - res/2
		if actual < 0 {
			actual = 0
		}
	}
	c.Stats.HP -= actual
	if c.Stats.HP <= 0 {
		c.Stats.HP = 0
		c.Alive = false
		fmt.Printf("%s погиб(ла)!\n", c.Name)
	}
}

func (c *Character) Heal(amount int) {
	c.Stats.HP += amount
	if c.Stats.HP > c.Stats.HPMax {
		c.Stats.HP = c.Stats.HPMax
	}
}

func (c *Character) UseMP(amount int) bool {
	if c.Stats.MP < amount {
		return false
	}
	c.Stats.MP -= amount
	return true
}

func (c *Character) AddEffect(e Effect) {
	c.Effects = append(c.Effects, e)
	fmt.Printf("%s получил эффект: %s (dur=%d)\n", c.Name, e.Name, e.Duration)
}

func (c *Character) EquipWeapon(w *Weapon) {
	c.Weapon = w
	fmt.Printf("%s экипировал оружие: %s\n", c.Name, w.Name)
}

func (c *Character) EquipArmor(a *Armor) {
	c.Armor = a
	// apply HP bonus immediately
	if a.HPBonus != 0 {
		c.Stats.HPMax += a.HPBonus
		c.Stats.HP += a.HPBonus
	}
	fmt.Printf("%s надел броню: %s\n", c.Name, a.Name)
}

func (c *Character) BasicAttack(target *Character) {
	if !c.Alive {
		return
	}
	min, max, dtype := 1, 2, Physical
	if c.Weapon != nil {
		min = c.Weapon.DamageMin
		max = c.Weapon.DamageMax
		dtype = c.Weapon.DamageType
	}
	base := rand.Intn(max-min+1) + min
	// apply attack stat
	base += c.EffectiveAttack()
	// crit?
	if rand.Float64() < c.Stats.CritRate {
		base = int(float64(base) * c.Stats.CritMult)
		fmt.Printf("Критический удар! (%s)\n", c.Name)
	}
	fmt.Printf("%s атакует %s на %d урона (%s)\n", c.Name, target.Name, base, dtype)
	target.TakeDamage(base, dtype)
}

func (c *Character) UseSkillAt(idx int, targets []*Character) {
	if idx < 0 || idx >= len(c.Skills) {
		fmt.Printf("%s попытался использовать несуществующий скилл\n", c.Name)
		return
	}
	s := c.Skills[idx]
	if c.Stats.MP < s.MPCost {
		fmt.Printf("%s не хватает MP для %s\n", c.Name, s.Name)
		return
	}
	c.UseMP(s.MPCost)
	fmt.Printf("%s использует умение %s\n", c.Name, s.Name)
	// damage
	if s.DamageMultiplier > 0 {
		for _, t := range targets {
			if !t.Alive {
				continue
			}
			var power int
			if s.DamageType == Magic {
				power = int(float64(c.Stats.Magic) * s.DamageMultiplier)
			} else {
				power = int(float64(c.EffectiveAttack()) * s.DamageMultiplier)
			}
			// randomness
			randAdd := rand.Intn(3) - 1
			power += randAdd
			// crit (using same critrate)
			if rand.Float64() < c.Stats.CritRate {
				power = int(float64(power) * c.Stats.CritMult)
				fmt.Printf("Крит умения! (%s)\n", c.Name)
			}
			fmt.Printf("%s наносит %d урона %s скиллом %s\n", c.Name, power, t.Name, s.Name)
			t.TakeDamage(power, s.DamageType)
		}
	}
	// heal
	if s.HealHP > 0 {
		for _, t := range targets {
			if !t.Alive {
				continue
			}
			t.Heal(s.HealHP)
			fmt.Printf("%s исцеляет %s на %d HP\n", c.Name, t.Name, s.HealHP)
		}
	}
}

// choose target helper: first alive in other team
func chooseFirstAlive(list []*Character) *Character {
	for _, c := range list {
		if c != nil && c.Alive {
			return c
		}
	}
	return nil
}

type Battle struct {
	Players []*Character
	Enemies []*Character
	Round   int
	Logs    []string
}

func NewBattle(players []*Character, enemies []*Character) *Battle {
	return &Battle{
		Players: players,
		Enemies: enemies,
		Round:   0,
		Logs:    []string{},
	}
}

func (b *Battle) AllDead(team string) bool {
	if team == "player" {
		for _, p := range b.Players {
			if p != nil && p.Alive {
				return false
			}
		}
		return true
	}
	for _, e := range b.Enemies {
		if e != nil && e.Alive {
			return false
		}
	}
	return true
}

func (b *Battle) Turn() {
	b.Round++
	fmt.Printf("=== Раунд %d ===\n", b.Round)
	// build action order by speed
	all := []*Character{}
	all = append(all, b.Players...)
	all = append(all, b.Enemies...)
	// simple sort by EffectiveSpeed (bubble for simplicity)
	for i := 0; i < len(all); i++ {
		for j := 0; j < len(all)-1; j++ {
			if all[j] == nil || all[j+1] == nil {
				continue
			}
			if all[j].EffectiveSpeed() < all[j+1].EffectiveSpeed() {
				all[j], all[j+1] = all[j+1], all[j]
			}
		}
	}
	// each acts if alive
	for _, actor := range all {
		if actor == nil || !actor.Alive {
			continue
		}
		actor.ApplyEffectsStartTurn()

		// choose action:
		if actor.Team == "player" {
			// simple AI: if have MP and skill, 30% chance use skill
			if len(actor.Skills) > 0 && actor.Stats.MP >= actor.Skills[0].MPCost && rand.Float64() < 0.3 {
				target := chooseFirstAlive(b.EnemiesAsSlice())
				if target != nil {
					actor.UseSkillAt(0, []*Character{target})
				}
			} else {
				// basic attack enemy
				target := chooseFirstAlive(b.EnemiesAsSlice())
				if target != nil {
					actor.BasicAttack(target)
				}
			}
		} else {
			// enemy AI: use skill if low hp to heal or otherwise attack
			if len(actor.Skills) > 0 && actor.Stats.HP < actor.Stats.HPMax/2 && actor.Stats.MP >= actor.Skills[0].MPCost {
				// try heal self
				actor.UseSkillAt(0, []*Character{actor})
			} else {
				target := chooseFirstAlive(b.PlayersAsSlice())
				if target != nil {
					actor.BasicAttack(target)
				}
			}
		}

		actor.ApplyEffectsEndTurn()
		// small sleep for readability
		time.Sleep(200 * time.Millisecond)
		// check for end
		if b.AllDead("enemy") || b.AllDead("player") {
			return
		}
	}
}

func (b *Battle) PlayersAsSlice() []*Character {
	out := make([]*Character, 0, len(b.Players))
	for _, p := range b.Players {
		if p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (b *Battle) EnemiesAsSlice() []*Character {
	out := make([]*Character, 0, len(b.Enemies))
	for _, p := range b.Enemies {
		if p != nil {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// create player characters
	heroStats := Stats{
		HPMax:    60,
		MPMax:    30,
		Attack:   6,
		Defense:  3,
		Magic:    4,
		Resist:   2,
		Speed:    7,
		CritRate: 0.12,
		CritMult: 1.7,
	}
	hero := NewCharacter("p1", "Герой", heroStats)
	skillFire := Skill{
		ID:               "s1",
		Name:             "Огненный шар",
		Description:      "Наносит магический урон",
		MPCost:           6,
		DamageMultiplier: 2.2,
		DamageType:       Magic,
	}
	hero.Skills = append(hero.Skills, skillFire)
	hero.EquipWeapon(&Weapon{
		Name:        "Меч новичка",
		DamageMin:   3,
		DamageMax:   6,
		DamageType:  Physical,
		AttackBonus: 1,
	})
	hero.EquipArmor(&Armor{
		Name:         "Кожаная броня",
		DefenseBonus: 1,
		ResistBonus:  0,
		HPBonus:      5,
	})

	// companion
	clericStats := Stats{
		HPMax:    45,
		MPMax:    50,
		Attack:   3,
		Defense:  2,
		Magic:    6,
		Resist:   3,
		Speed:    5,
		CritRate: 0.05,
		CritMult: 1.5,
	}
	cleric := NewCharacter("p2", "Жрец", clericStats)
	healSkill := Skill{
		ID:         "he1",
		Name:       "Исцеление",
		MPCost:     8,
		HealHP:     18,
		DamageType: Magic,
	}
	cleric.Skills = append(cleric.Skills, healSkill)
	cleric.EquipWeapon(&Weapon{
		Name:       "Посох",
		DamageMin:  1,
		DamageMax:  3,
		DamageType: Magic,
		MagicBonus: 1,
	})

	// enemies
	goblinStats := Stats{
		HPMax:    20,
		MPMax:    5,
		Attack:   4,
		Defense:  1,
		Magic:    1,
		Resist:   0,
		Speed:    6,
		CritRate: 0.06,
		CritMult: 1.5,
	}
	gob1 := NewCharacter("e1", "Гоблин-1", goblinStats)
	gob1.EquipWeapon(&Weapon{Name: "Короткий кинжал", DamageMin: 2, DamageMax: 4, DamageType: Physical})
	gob2 := NewCharacter("e2", "Гоблин-2", goblinStats)
	gob2.EquipWeapon(&Weapon{Name: "Короткий кинжал", DamageMin: 2, DamageMax: 4, DamageType: Physical})

	orcStats := Stats{
		HPMax:    35,
		MPMax:    0,
		Attack:   8,
		Defense:  4,
		Magic:    0,
		Resist:   1,
		Speed:    4,
		CritRate: 0.08,
		CritMult: 1.4,
	}
	orc := NewCharacter("e3", "Орк", orcStats)
	orc.EquipWeapon(&Weapon{Name: "Клевец", DamageMin: 5, DamageMax: 8, DamageType: Physical})

	// assign teams
	hero.Team = "player"
	cleric.Team = "player"
	gob1.Team = "enemy"
	gob2.Team = "enemy"
	orc.Team = "enemy"

	players := []*Character{hero, cleric}
	enemies := []*Character{gob1, gob2, orc}

	b := NewBattle(players, enemies)

	// run battle until one side died
	for {
		if b.AllDead("enemy") {
			fmt.Println("Игроки победили!")
			break
		}
		if b.AllDead("player") {
			fmt.Println("Враги победили...")
			break
		}
		b.Turn()
	}
}
