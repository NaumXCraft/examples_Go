package main

import (
	"fmt"
	"math/rand"
	"sort"
)

// RPG — turn-based battle simulator.
// Grok (Запуск: go run main.go)

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
	CritRate float64
	CritMult float64
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
	Consumable  bool
	HealHP      int
	HealMP      int
	EquipSlot   string
	Weapon      *Weapon
	Armor       *Armor
}

type Weapon struct {
	Name        string
	DamageMin   int
	DamageMax   int
	DamageType  DamageType
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
	ID               string
	Name             string
	Description      string
	MPCost           int
	DamageMultiplier float64
	DamageType       DamageType
	HealHP           int
	TargetAll        bool
	Effect           *Effect // Optional effect to apply on targets
}

type Effect struct {
	ID       string
	Name     string
	Duration int
	AtkMod   int
	DefMod   int
	SpeedMod int
	DotHP    int
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
	Alive   bool
	Team    string
}

func NewCharacter(id, name, team string, baseStats Stats) *Character {
	s := baseStats.Clone()
	s.HP = s.HPMax
	s.MP = s.MPMax
	return &Character{
		ID:      id,
		Name:    name,
		Team:    team,
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

func (c *Character) ApplyEffectsStartTurn(logs *[]string) {
	totalDot := 0
	for _, e := range c.Effects {
		if e.DotHP != 0 {
			totalDot += e.DotHP
		}
	}
	if totalDot != 0 {
		c.TakeDamage(totalDot, Pure, logs)
		*logs = append(*logs, fmt.Sprintf("%s takes %d DOT damage", c.Name, totalDot))
	}
}

func (c *Character) ApplyEffectsEndTurn(logs *[]string) {
	newEffects := make([]Effect, 0, len(c.Effects))
	for i := range c.Effects {
		c.Effects[i].Tick()
		if c.Effects[i].Duration > 0 {
			newEffects = append(newEffects, c.Effects[i])
		} else {
			*logs = append(*logs, fmt.Sprintf("Effect %s on %s ended", c.Effects[i].Name, c.Name))
		}
	}
	c.Effects = newEffects
}

func (c *Character) TakeDamage(amount int, dtype DamageType, logs *[]string) {
	actual := amount
	if dtype == Physical {
		def := c.EffectiveDefense()
		actual = amount - def/2
	} else if dtype == Magic {
		res := c.Stats.Resist
		actual = amount - res/2
	}
	if actual < 1 {
		actual = 1 // Min damage
	}
	c.Stats.HP -= actual
	if c.Stats.HP <= 0 {
		c.Stats.HP = 0
		c.Alive = false
		*logs = append(*logs, fmt.Sprintf("%s died!", c.Name))
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

func (c *Character) AddEffect(e Effect, logs *[]string) {
	c.Effects = append(c.Effects, e)
	*logs = append(*logs, fmt.Sprintf("%s gains effect: %s (dur=%d)", c.Name, e.Name, e.Duration))
}

func (c *Character) EquipWeapon(w *Weapon, logs *[]string) {
	c.Weapon = w
	*logs = append(*logs, fmt.Sprintf("%s equips weapon: %s", c.Name, w.Name))
}

func (c *Character) UnequipWeapon() {
	c.Weapon = nil
}

func (c *Character) EquipArmor(a *Armor, logs *[]string) {
	c.Armor = a
	if a.HPBonus != 0 {
		c.Stats.HPMax += a.HPBonus
		c.Stats.HP += a.HPBonus // Apply bonus
	}
	*logs = append(*logs, fmt.Sprintf("%s equips armor: %s", c.Name, a.Name))
}

func (c *Character) UnequipArmor() {
	if c.Armor != nil && c.Armor.HPBonus != 0 {
		c.Stats.HPMax -= c.Armor.HPBonus
		if c.Stats.HP > c.Stats.HPMax {
			c.Stats.HP = c.Stats.HPMax
		}
	}
	c.Armor = nil
}

func (c *Character) BasicAttack(target *Character, logs *[]string) {
	if !c.Alive {
		return
	}
	min_, max_, dtype := 1, 2, Physical
	if c.Weapon != nil {
		min_ = c.Weapon.DamageMin
		max_ = c.Weapon.DamageMax
		dtype = c.Weapon.DamageType
	}
	base := rand.Intn(max_-min_+1) + min_ + c.EffectiveAttack()
	if rand.Float64() < c.Stats.CritRate {
		base = int(float64(base) * c.Stats.CritMult)
		*logs = append(*logs, fmt.Sprintf("Critical hit! (%s)", c.Name))
	}
	*logs = append(*logs, fmt.Sprintf("%s attacks %s for %d damage (%s)", c.Name, target.Name, base, dtype))
	target.TakeDamage(base, dtype, logs)
}

func (c *Character) UseSkillAt(idx int, targets []*Character, logs *[]string) {
	if idx < 0 || idx >= len(c.Skills) {
		*logs = append(*logs, fmt.Sprintf("%s tried to use invalid skill", c.Name))
		return
	}
	s := c.Skills[idx]
	if !c.UseMP(s.MPCost) {
		*logs = append(*logs, fmt.Sprintf("%s lacks MP for %s", c.Name, s.Name))
		return
	}
	*logs = append(*logs, fmt.Sprintf("%s uses skill %s", c.Name, s.Name))
	// Damage
	if s.DamageMultiplier > 0 {
		for _, t := range targets {
			if !t.Alive {
				continue
			}
			power := int(float64(c.EffectiveAttack()) * s.DamageMultiplier)
			if s.DamageType == Magic {
				power = int(float64(c.Stats.Magic) * s.DamageMultiplier)
			}
			power += rand.Intn(3) - 1
			if rand.Float64() < c.Stats.CritRate {
				power = int(float64(power) * c.Stats.CritMult)
				*logs = append(*logs, fmt.Sprintf("Skill crit! (%s)", c.Name))
			}
			*logs = append(*logs, fmt.Sprintf("%s deals %d damage to %s with %s", c.Name, power, t.Name, s.Name))
			t.TakeDamage(power, s.DamageType, logs)
			if s.Effect != nil {
				t.AddEffect(*s.Effect, logs)
			}
		}
	}
	// Heal
	if s.HealHP > 0 {
		for _, t := range targets {
			if !t.Alive {
				continue
			}
			t.Heal(s.HealHP)
			*logs = append(*logs, fmt.Sprintf("%s heals %s for %d HP", c.Name, t.Name, s.HealHP))
			if s.Effect != nil {
				t.AddEffect(*s.Effect, logs)
			}
		}
	}
}

func chooseFirstAlive(list []*Character) *Character {
	for _, c := range list {
		if c.Alive {
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
	var chars []*Character
	if team == "player" {
		chars = b.Players
	} else {
		chars = b.Enemies
	}
	for _, c := range chars {
		if c.Alive {
			return false
		}
	}
	return true
}

func (b *Battle) Turn() {
	b.Round++
	b.Logs = append(b.Logs, fmt.Sprintf("=== Round %d ===", b.Round))
	all := append([]*Character{}, b.Players...)
	all = append(all, b.Enemies...)
	// Sort by speed descending
	sort.Slice(all, func(i, j int) bool {
		return all[i].EffectiveSpeed() > all[j].EffectiveSpeed()
	})

	for _, actor := range all {
		if !actor.Alive {
			continue
		}
		actor.ApplyEffectsStartTurn(&b.Logs)

		var targets []*Character
		if actor.Team == "player" {
			targets = b.Enemies
		} else {
			targets = b.Players
		}

		usedAction := false
		// Improved AI: 50% chance to use random skill if possible, else basic attack
		if len(actor.Skills) > 0 && rand.Float64() < 0.5 {
			// Choose random skill with enough MP
			skillIdx := rand.Intn(len(actor.Skills))
			s := actor.Skills[skillIdx]
			if actor.Stats.MP >= s.MPCost {
				targ := []*Character{chooseFirstAlive(targets)}
				if s.HealHP > 0 || (actor.Stats.HP < actor.Stats.HPMax/2 && actor.Team != "player") {
					targ = []*Character{actor} // Self-heal if low HP for enemies
				}
				if s.TargetAll {
					targ = targets
				}
				actor.UseSkillAt(skillIdx, targ, &b.Logs)
				usedAction = true
			}
		}

		if !usedAction {
			// Basic attack
			target := chooseFirstAlive(targets)
			if target != nil {
				actor.BasicAttack(target, &b.Logs)
			}
		}

		actor.ApplyEffectsEndTurn(&b.Logs)

		if b.AllDead("player") || b.AllDead("enemy") {
			return
		}
	}
}

func (b *Battle) Run() {
	for !b.AllDead("player") && !b.AllDead("enemy") {
		b.Turn()
	}
	if b.AllDead("enemy") {
		b.Logs = append(b.Logs, "Players win!")
	} else {
		b.Logs = append(b.Logs, "Enemies win!")
	}
}

func main() {
	//rand.Seed(time.Now().UnixNano())

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
	hero := NewCharacter("p1", "Герой", "player", heroStats)
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
	}, &[]string{})
	hero.EquipArmor(&Armor{
		Name:         "Кожаная броня",
		DefenseBonus: 1,
		ResistBonus:  0,
		HPBonus:      5,
	}, &[]string{})

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
	cleric := NewCharacter("p2", "Жрец", "player", clericStats)
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
	}, &[]string{})

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
	gob1 := NewCharacter("e1", "Гоблин-1", "enemy", goblinStats)
	gob1.EquipWeapon(&Weapon{Name: "Короткий кинжал", DamageMin: 2, DamageMax: 4, DamageType: Physical}, &[]string{})
	gob2 := NewCharacter("e2", "Гоблин-2", "enemy", goblinStats)
	gob2.EquipWeapon(&Weapon{Name: "Короткий кинжал", DamageMin: 2, DamageMax: 4, DamageType: Physical}, &[]string{})

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
	orc := NewCharacter("e3", "Орк", "enemy", orcStats)
	orc.EquipWeapon(&Weapon{Name: "Клевец", DamageMin: 5, DamageMax: 8, DamageType: Physical}, &[]string{})

	players := []*Character{hero, cleric}
	enemies := []*Character{gob1, gob2, orc}

	b := NewBattle(players, enemies)
	b.Run()

	// Print logs
	for _, log := range b.Logs {
		fmt.Println(log)
	}
}
