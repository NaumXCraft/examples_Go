// file: cardgame.go
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
Card Duel / Deck-building prototype
Enhanced controls (commands):
 - play <i>        : play card index i from hand (0-based)
 - play <i> <t>    : play card i targeting player t (0 = you, 1 = enemy) — for clarity (target optional)
 - end             : end turn
 - info            : show brief state
 - hand            : show hand (same as info but focused)
 - deck            : show deck count
 - discard         : show discard count
 - exhaust         : show exhaust count
 - inspect <i>     : show full details of card in hand index i
 - draw <n>        : draw n cards (debug)
 - shuffle         : shuffle your deck (debug)
 - discardc <i>    : move card i from hand to discard
 - exhaustc <i>    : move card i from hand to exhaust
 - mulligan        : discard current hand, draw same number (reshuffle)
 - concede         : resign (end game)
 - help / h        : show help
 - quit / q        : exit program
*/

type CardType string

const (
	AttackCard CardType = "attack"
	SkillCard  CardType = "skill"
	PowerCard  CardType = "power"
)

type CardEffect struct {
	Damage    int // damage to target
	Block     int // give block to self
	Draw      int // draw cards
	Heal      int // heal self
	ApplyBurn int // burn damage per turn
	ApplyWeak int // example: reduce attack (not used heavily)
	ApplyVuln int // example: take more damage
}

type Card struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        CardType   `json:"type"`
	Cost        int        `json:"cost"`
	Attack      int        `json:"attack"`
	Effect      CardEffect `json:"effect"`
	Description string     `json:"description"`
	Rarity      string     `json:"rarity"`
	Exhausts    bool       `json:"exhausts"`
}

type Status struct {
	Name     string
	Duration int
	Value    int // e.g., burn damage per turn
}

type Player struct {
	Name      string
	HP        int
	MaxHP     int
	Energy    int
	MaxEnergy int
	Block     int
	Deck      []Card
	Hand      []Card
	Discard   []Card
	Exhaust   []Card
	Statuses  []Status
	IsAI      bool
}

// GameState holds turn info and players
type GameState struct {
	Turn      int
	Players   [2]*Player // player 0 = human, 1 = enemy
	ActiveIdx int        // whose turn (0 or 1)
	Rand      *rand.Rand
}

// Utility helpers
func shuffleCards(r *rand.Rand, cards []Card) []Card {
	out := make([]Card, len(cards))
	copy(out, cards)
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func drawOne(p *Player, gs *GameState) {
	if len(p.Deck) == 0 {
		// reshuffle discard into deck
		if len(p.Discard) == 0 {
			return
		}
		p.Deck = shuffleCards(gs.Rand, p.Discard)
		p.Discard = nil
	}
	// draw top
	card := p.Deck[0]
	p.Deck = p.Deck[1:]
	p.Hand = append(p.Hand, card)
}

func drawN(p *Player, gs *GameState, n int) {
	for i := 0; i < n; i++ {
		drawOne(p, gs)
	}
}

func applyStartStatuses(p *Player) int {
	total := 0
	newStatuses := make([]Status, 0, len(p.Statuses))
	for _, s := range p.Statuses {
		if s.Name == "Burn" && s.Value > 0 {
			fmt.Printf("%s получает %d урона от %s\n", p.Name, s.Value, s.Name)
			total += s.Value
		}
		s.Duration--
		if s.Duration > 0 {
			newStatuses = append(newStatuses, s)
		} else {
			fmt.Printf("Эффект %s на %s завершился\n", s.Name, p.Name)
		}
	}
	p.Statuses = newStatuses
	return total
}

func (p *Player) takeDamage(d int) {
	if p.Block >= d {
		p.Block -= d
		fmt.Printf("%s блокирует %d урона (осталось блок: %d)\n", p.Name, d, p.Block)
		return
	}
	d -= p.Block
	p.Block = 0
	p.HP -= d
	fmt.Printf("%s получает %d урона (HP=%d)\n", p.Name, d, p.HP)
	if p.HP < 0 {
		p.HP = 0
	}
}

func (p *Player) heal(h int) {
	p.HP += h
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
	fmt.Printf("%s исцеляется на %d (HP=%d)\n", p.Name, h, p.HP)
}

func (gs *GameState) playCard(playerIdx int, handIdx int, targetIdx int) error {
	p := gs.Players[playerIdx]
	if handIdx < 0 || handIdx >= len(p.Hand) {
		return fmt.Errorf("invalid hand index")
	}
	card := p.Hand[handIdx]
	if card.Cost > p.Energy {
		return fmt.Errorf("not enough energy")
	}
	// consume energy
	p.Energy -= card.Cost
	// effects apply
	target := gs.Players[targetIdx]
	if card.Attack > 0 {
		fmt.Printf("%s наносит %d урона (карта %s)\n", p.Name, card.Attack, card.Name)
		target.takeDamage(card.Attack)
	}
	if card.Effect.Damage > 0 {
		fmt.Printf("%s наносит эффектом %d урона\n", p.Name, card.Effect.Damage)
		target.takeDamage(card.Effect.Damage)
	}
	if card.Effect.Block > 0 {
		p.Block += card.Effect.Block
		fmt.Printf("%s получает %d блока (total block %d)\n", p.Name, card.Effect.Block, p.Block)
	}
	if card.Effect.Draw > 0 {
		fmt.Printf("%s добирает %d карт\n", p.Name, card.Effect.Draw)
		drawN(p, gs, card.Effect.Draw)
	}
	if card.Effect.Heal > 0 {
		p.heal(card.Effect.Heal)
	}
	if card.Effect.ApplyBurn > 0 {
		target.Statuses = append(target.Statuses, Status{Name: "Burn", Duration: 3, Value: card.Effect.ApplyBurn})
		fmt.Printf("%s накладывает Burn %d на %s\n", p.Name, card.Effect.ApplyBurn, target.Name)
	}
	// exhaust or discard
	if card.Exhausts {
		p.Exhaust = append(p.Exhaust, card)
	} else {
		p.Discard = append(p.Discard, card)
	}
	// remove from hand
	p.Hand = append(p.Hand[:handIdx], p.Hand[handIdx+1:]...)
	return nil
}

func (gs *GameState) aiTakeTurn(aiIdx int) {
	p := gs.Players[aiIdx]
	oppIdx := 1 - aiIdx
	// try to play cards while possible
	played := true
	for played {
		played = false
		// prefer attacks/burn/block
		for i := 0; i < len(p.Hand); i++ {
			c := p.Hand[i]
			if c.Cost <= p.Energy {
				if c.Attack > 0 || c.Effect.Damage > 0 || c.Effect.ApplyBurn > 0 || c.Effect.Block > 0 {
					fmt.Printf("AI %s играет карту %s\n", p.Name, c.Name)
					_ = gs.playCard(aiIdx, i, oppIdx)
					played = true
					break
				}
			}
		}
	}
}

func (gs *GameState) startTurn(idx int) {
	p := gs.Players[idx]
	fmt.Printf("\n--- %s ход ---\n", p.Name)
	gs.Turn++
	p.Energy = p.MaxEnergy
	// reset block at start of turn for that player
	p.Block = 0
	// draw up to 5
	drawN(p, gs, 5-len(p.Hand))
	// apply statuses
	dmg := applyStartStatuses(p)
	if dmg > 0 {
		p.takeDamage(dmg)
	}
}

func (gs *GameState) endTurn(idx int) {
	p := gs.Players[idx]
	// move hand to discard (unless exhaust or powers)
	p.Discard = append(p.Discard, p.Hand...)
	p.Hand = nil
}

func (gs *GameState) checkWin() (bool, string) {
	p0 := gs.Players[0]
	p1 := gs.Players[1]
	if p0.HP <= 0 {
		return true, fmt.Sprintf("%s проиграл", p0.Name)
	}
	if p1.HP <= 0 {
		return true, fmt.Sprintf("%s выиграл!", p0.Name)
	}
	return false, ""
}

func starterDeck() []Card {
	return []Card{
		{ID: "c_atk1", Name: "Удар", Type: AttackCard, Cost: 1, Attack: 6, Description: "Простой удар", Rarity: "common"},
		{ID: "c_block1", Name: "Блок", Type: SkillCard, Cost: 1, Effect: CardEffect{Block: 6}, Description: "Получить блок", Rarity: "common"},
		{ID: "c_strike", Name: "Огненный удар", Type: AttackCard, Cost: 2, Effect: CardEffect{Damage: 8, ApplyBurn: 2}, Description: "Урон + накладывает Burn", Rarity: "uncommon"},
		{ID: "c_draw", Name: "Вдохновение", Type: SkillCard, Cost: 0, Effect: CardEffect{Draw: 2}, Description: "Добрать 2 карты", Rarity: "common"},
		{ID: "c_heal", Name: "Исцеление", Type: SkillCard, Cost: 2, Effect: CardEffect{Heal: 5}, Description: "Исцеление", Rarity: "rare", Exhausts: true},
		{ID: "c_atk1", Name: "Удар", Type: AttackCard, Cost: 1, Attack: 6, Description: "Простой удар", Rarity: "common"},
		{ID: "c_block1", Name: "Блок", Type: SkillCard, Cost: 1, Effect: CardEffect{Block: 6}, Description: "Получить блок", Rarity: "common"},
		{ID: "c_atk1", Name: "Удар", Type: AttackCard, Cost: 1, Attack: 6},
	}
}

func cloneDeck(d []Card) []Card {
	out := make([]Card, len(d))
	copy(out, d)
	return out
}

func newPlayer(name string, maxhp, maxenergy int, deck []Card, isAI bool, r *rand.Rand) *Player {
	p := &Player{
		Name:      name,
		MaxHP:     maxhp,
		HP:        maxhp,
		MaxEnergy: maxenergy,
		Energy:    maxenergy,
		Block:     0,
		Deck:      shuffleCards(r, deck),
		Hand:      nil,
		Discard:   nil,
		Exhaust:   nil,
		Statuses:  nil,
		IsAI:      isAI,
	}
	// draw opening hand
	for i := 0; i < 5; i++ {
		drawOne(p, &GameState{Rand: r})
	}
	return p
}

// helpers for debug / commands
func showInfo(gs *GameState) {
	p := gs.Players[gs.ActiveIdx]
	opp := gs.Players[1-gs.ActiveIdx]
	fmt.Printf("\n=== %s === Turn:%d Active:%d\n", p.Name, gs.Turn, gs.ActiveIdx)
	fmt.Printf("%s HP:%d/%d E:%d Block:%d Deck:%d Discard:%d Exhaust:%d Hand:%d\n",
		p.Name, p.HP, p.MaxHP, p.Energy, p.Block, len(p.Deck), len(p.Discard), len(p.Exhaust), len(p.Hand))
	fmt.Printf("Opponent %s HP:%d/%d Block:%d\n", opp.Name, opp.HP, opp.MaxHP, opp.Block)
	fmt.Println("Hand:")
	for i, c := range p.Hand {
		fmt.Printf(" [%d] %s (cost:%d) - %s\n", i, c.Name, c.Cost, c.Description)
	}
}

func showDeckCounts(p *Player) {
	fmt.Printf("Deck:%d Discard:%d Exhaust:%d Hand:%d\n", len(p.Deck), len(p.Discard), len(p.Exhaust), len(p.Hand))
}

func inspectCard(p *Player, idx int) {
	if idx < 0 || idx >= len(p.Hand) {
		fmt.Println("invalid index")
		return
	}
	c := p.Hand[idx]
	fmt.Println("----- Card Details -----")
	fmt.Printf("Name: %s (%s)\n", c.Name, c.Rarity)
	fmt.Printf("Type: %s Cost: %d Attack:%d Exhausts:%v\n", c.Type, c.Cost, c.Attack, c.Exhausts)
	fmt.Printf("Effect: Damage=%d Block=%d Draw=%d Heal=%d Burn=%d\n",
		c.Effect.Damage, c.Effect.Block, c.Effect.Draw, c.Effect.Heal, c.Effect.ApplyBurn)
	fmt.Printf("Desc: %s\n", c.Description)
	fmt.Println("------------------------")
}

func discardCardFromHand(p *Player, idx int) {
	if idx < 0 || idx >= len(p.Hand) {
		fmt.Println("invalid index")
		return
	}
	c := p.Hand[idx]
	p.Discard = append(p.Discard, c)
	p.Hand = append(p.Hand[:idx], p.Hand[idx+1:]...)
	fmt.Printf("Карта %s перемещена в сброс\n", c.Name)
}

func exhaustCardFromHand(p *Player, idx int) {
	if idx < 0 || idx >= len(p.Hand) {
		fmt.Println("invalid index")
		return
	}
	c := p.Hand[idx]
	p.Exhaust = append(p.Exhaust, c)
	p.Hand = append(p.Hand[:idx], p.Hand[idx+1:]...)
	fmt.Printf("Карта %s исчерпана (exhaust)\n", c.Name)
}

func mulligan(p *Player, gs *GameState) {
	handSize := len(p.Hand)
	p.Discard = append(p.Discard, p.Hand...)
	p.Hand = nil
	drawN(p, gs, handSize)
	fmt.Println("Mulligan выполнен: новая рука.")
}

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	playerDeck := cloneDeck(starterDeck())
	enemyDeck := cloneDeck(starterDeck())

	player := newPlayer("Игрок", 30, 3, playerDeck, false, r)
	enemy := newPlayer("Враг", 28, 3, enemyDeck, true, r)

	gs := &GameState{
		Turn:      0,
		Players:   [2]*Player{player, enemy},
		ActiveIdx: 0,
		Rand:      r,
	}

	reader := bufio.NewReader(os.Stdin)
mainloop:
	for {
		active := gs.ActiveIdx
		opp := 1 - active
		gs.startTurn(active)
		if gs.Players[active].IsAI {
			gs.aiTakeTurn(active)
			gs.endTurn(active)
			gs.ActiveIdx = opp
		} else {
			// human interactive loop
			for {
				fmt.Printf("\n-- %s -- HP:%d/%d E:%d Block:%d\n", gs.Players[active].Name, gs.Players[active].HP, gs.Players[active].MaxHP, gs.Players[active].Energy, gs.Players[active].Block)
				fmt.Print("Команда (h help): ")
				line, _ := reader.ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.Split(line, " ")
				cmd := strings.ToLower(parts[0])
				switch cmd {
				case "play":
					if len(parts) < 2 {
						fmt.Println("play <handIndex> [target]")
						continue
					}
					i, err := strconv.Atoi(parts[1])
					if err != nil {
						fmt.Println("invalid index")
						continue
					}
					target := opp
					if len(parts) >= 3 {
						t, err := strconv.Atoi(parts[2])
						if err == nil && (t == 0 || t == 1) {
							target = t
						}
					}
					if i < 0 || i >= len(gs.Players[active].Hand) {
						fmt.Println("invalid hand index")
						continue
					}
					err = gs.playCard(active, i, target)
					if err != nil {
						fmt.Println("Не удалось сыграть карту:", err)
					} else {
						if ok, res := gs.checkWin(); ok {
							fmt.Println(res)
							break mainloop
						}
					}
				case "end":
					gs.endTurn(active)
					gs.ActiveIdx = opp
					break
				case "info":
					showInfo(gs)
				case "hand":
					p := gs.Players[active]
					fmt.Println("Hand:")
					for i, c := range p.Hand {
						fmt.Printf(" [%d] %s (cost:%d) - %s\n", i, c.Name, c.Cost, c.Description)
					}
				case "deck":
					showDeckCounts(gs.Players[active])
				case "discard":
					fmt.Printf("Discard count: %d\n", len(gs.Players[active].Discard))
				case "exhaust":
					fmt.Printf("Exhaust count: %d\n", len(gs.Players[active].Exhaust))
				case "inspect":
					if len(parts) < 2 {
						fmt.Println("inspect <handIndex>")
						continue
					}
					i, err := strconv.Atoi(parts[1])
					if err != nil {
						fmt.Println("invalid index")
						continue
					}
					inspectCard(gs.Players[active], i)
				case "draw":
					n := 1
					if len(parts) >= 2 {
						if v, err := strconv.Atoi(parts[1]); err == nil {
							n = v
						}
					}
					drawN(gs.Players[active], gs, n)
					fmt.Printf("Нарисовано %d карт\n", n)
				case "shuffle":
					gs.Players[active].Deck = shuffleCards(gs.Rand, gs.Players[active].Deck)
					fmt.Println("Deck shuffled.")
				case "discardc":
					if len(parts) < 2 {
						fmt.Println("discardc <handIndex>")
						continue
					}
					i, err := strconv.Atoi(parts[1])
					if err != nil {
						fmt.Println("invalid index")
						continue
					}
					discardCardFromHand(gs.Players[active], i)
				case "exhaustc":
					if len(parts) < 2 {
						fmt.Println("exhaustc <handIndex>")
						continue
					}
					i, err := strconv.Atoi(parts[1])
					if err != nil {
						fmt.Println("invalid index")
						continue
					}
					exhaustCardFromHand(gs.Players[active], i)
				case "mulligan":
					mulligan(gs.Players[active], gs)
				case "concede":
					fmt.Println("Вы сдались.")
					break mainloop
				case "help", "h":
					fmt.Println("Доступные команды:")
					fmt.Println(" play <i> [target]  - сыграть карту из руки (target: 0=you,1=enemy) ")
					fmt.Println(" end                - закончить ход")
					fmt.Println(" info               - показать состояние")
					fmt.Println(" hand               - показать руку")
					fmt.Println(" deck               - показать колоду/сброс/исчерпанные размеры")
					fmt.Println(" inspect <i>        - показать детали карты из руки")
					fmt.Println(" draw <n>           - добрать n карт (debug)")
					fmt.Println(" shuffle            - перетасовать колоду")
					fmt.Println(" discardc <i>       - отправить карту i из руки в сброс")
					fmt.Println(" exhaustc <i>       - отправить карту i из руки в exhaust")
					fmt.Println(" mulligan           - сбросить руку и взять новые карты (debug)")
					fmt.Println(" concede            - сдаться")
					fmt.Println(" help / h           - показать эту справку")
					fmt.Println(" quit / q           - выйти")
				case "quit", "q":
					fmt.Println("Выход")
					return
				default:
					fmt.Println("Неизвестная команда. Введите 'h' для помощи.")
				}
				// after each action, check win
				if ok, res := gs.checkWin(); ok {
					fmt.Println(res)
					break mainloop
				}
			}
		}
		// after opponent turn check win
		if ok, res := gs.checkWin(); ok {
			fmt.Println(res)
			break
		}
	}
	fmt.Println("Игра завершена")
}
