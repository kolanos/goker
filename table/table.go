package table

import (
	"errors"
	"sort"

	"github.com/kolanos/goker/hand"
)

type Status int

const (
	Waiting Status = iota
	Dealing
)

type Round int

const (
	PreFlop Round = iota
	Flop
	Turn
	River
)

type Variant int

const (
	TexasHoldem Variant = iota
	OmahaHi
)

type LimitType int

const (
	Limit LimitType = iota
	NoLimit
	PotLimit
)

type Options struct {
	BuyIn   int       `json:"buyIn"`
	Variant Variant   `json:"variant"`
	Stakes  Stakes    `json:"stakes"`
	Limit   LimitType `json:"limit"`
}

type Stakes struct {
	BigBlind   int `json:"bigBlind"`
	SmallBlind int `json:"smallBlind"`
	Ante       int `json:"ante"`
}

type Table struct {
	options Options
	seats   []*Player
	dealer  hand.Dealer
	deck    *hand.Deck
	cards   []hand.Card
	active  *Player
	status  Status
	round   Round
	button  int
	cost    int
}

// New creates a new table
func New(dealer hand.Dealer, opts Options, numSeats int, playerIDs []string) *Table {
	status := Dealing
	if len(playerIDs) < 2 {
		status = Waiting
	}
	seats := make([]*Player, numSeats)
	t := &Table{
		options: opts,
		seats:   seats,
		round:   PreFlop,
		status:  status,
		dealer:  dealer,
	}
	for i, id := range playerIDs {
		p := &Player{
			ID:    id,
			Chips: opts.BuyIn,
		}
		t.Sit(p, i)
	}
	t.setupRound()
	return t
}

type State struct {
	Options Options     `json:"options"`
	Seats   []*Player   `json:"seats"`
	Cards   []hand.Card `json:"cards"`
	Active  *Player     `json:"active"`
	Status  Status      `json:"status"`
	Round   Round       `json:"round"`
	Button  int         `json:"button"`
	Cost    int         `json:"cost"`
	Pot     int         `json:"pot"`
}

func (t *Table) State() *State {
	var (
		seats []*Player
		pot   int
	)
	for _, seat := range t.seats {
		seats = append(seats, seat)
		if seat != nil {
			pot += seat.ChipsInPot
		}
	}
	return &State{
		Options: t.options,
		Seats:   seats,
		Cards:   append([]hand.Card(nil), t.cards...),
		Active:  t.active,
		Button:  t.button,
		Cost:    t.cost,
		Round:   t.round,
		Status:  t.status,
		Pot:     pot,
	}
}

type Action struct {
	Type  ActionType `json:"type"`
	Chips int        `json:"chips"`
}

type ActionType int

const (
	Fold ActionType = iota
	Check
	Call
	Bet
	Raise
	AllIn
)

// Sit seats a player at the table
func (t *Table) Sit(p *Player, seat int) error {
	if seat < -0 || seat > len(t.seats)-1 {
		return errors.New("Invalid seat selection")
	}
	if t.seats[seat] != nil {
		return errors.New("Seat is already taken")
	}
	t.seats[seat] = p
	p.Seat = seat
	return nil
}

// Fold folds hand for the active player
func (t *Table) Fold() error {
	return t.Act(Action{Type: Fold})
}

func (t *Table) Check() error {
	return t.Act(Action{Type: Check})
}

func (t *Table) Call() error {
	return t.Act(Action{Type: Call})
}

func (t *Table) Bet(chips int) error {
	return t.Act(Action{Type: Bet, Chips: chips})
}

func (t *Table) Raise(chips int) error {
	return t.Act(Action{Type: Raise, Chips: chips})
}

func (t *Table) AllIn() error {
	return t.Act(Action{Type: AllIn})
}

func (t *Table) Act(a Action) error {
	if t.active == nil {
		return errors.New("No active player")
	}
	if includes(t.LegalActions(), a.Type) == false {
		return errors.New("table: illegal action attempted")
	}
	switch a.Type {
	case Fold:
		t.active.Folded = true
	case Check:
	case Call:
		t.active.contribute(t.owed())
	case Bet, Raise:
		if a.Chips < t.options.Stakes.BigBlind {
			a.Chips = t.options.Stakes.BigBlind
		}
		if t.options.Limit == Limit {
			a.Chips = t.options.Stakes.BigBlind
			if t.round == Turn || t.round == River {
				a.Chips = a.Chips * 2
			}
		}
		chipsInPots := t.chipsInPots()
		if t.options.Limit == PotLimit && a.Chips > t.chipsInPots() {
			a.Chips = chipsInPots
		}
		t.active.contribute(t.owed())
		t.active.contribute(a.Chips)
		t.resetAction()
	case AllIn:
		t.active.contribute(t.owed())
		t.active.contribute(t.active.Chips)
		t.resetAction()
	}
	t.active.Acted = true
	if t.active.ChipsInPot > t.cost {
		t.cost = t.active.ChipsInPot
	}
	t.update()
	return nil
}

func (t *Table) Seats() []*Player {
	var seats []*Player
	for _, seat := range t.seats {
		seats = append(seats, seat)
	}
	return seats
}

func (t *Table) LegalActions() []ActionType {
	if t.owed() == 0 {
		return []ActionType{Fold, Check, Bet, AllIn}
	}
	if t.owed() > t.active.Chips {
		return []ActionType{Fold, Call}
	}
	return []ActionType{Fold, Call, Raise, AllIn}
}

func (t *Table) update() {
	seat := t.nextToAct()
	if seat != -1 {
		t.active = t.seats[seat]
		return
	}
	if len(t.contesting()) == 1 || t.round == River {
		t.payout()
		t.round = PreFlop
	} else {
		t.round = (t.round + 1) % (River + 1)
	}
	t.setupRound()
}

func (t *Table) Active() *Player {
	return t.active
}

func (t *Table) setupRound() {
	for _, seat := range t.seats {
		if seat != nil {
			seat.Acted = false
		}
	}
	switch t.round {
	case PreFlop:
		t.button = t.nextSeat(t.button)
		sb := t.nextSeat(t.button)
		bb := t.nextSeat(sb)
		if t.occupiedSeats() == 2 {
			sb = t.button
			bb = t.nextSeat(t.button)
		}
		t.deck = t.dealer.Deck()
		for _, seat := range t.seats {
			if seat != nil {
				seat.Cards = t.deck.PopMulti(2)
				seat.ChipsInPot = 0
				seat.Acted = false
				seat.Folded = false
				seat.AllIn = false
				seat.contribute(t.options.Stakes.Ante)
			}
		}
		t.seats[sb].contribute(t.options.Stakes.SmallBlind)
		t.seats[bb].contribute(t.options.Stakes.BigBlind)
		action := t.nextSeat(bb)
		t.active = t.seats[action]
		t.cost = t.options.Stakes.BigBlind
	case Flop:
		t.cards = t.deck.PopMulti(3)
		action := t.nextSeat(t.button)
		t.active = t.seats[action]
	case Turn, River:
		t.cards = append(t.cards, t.deck.Pop())
		action := t.nextSeat(t.button)
		t.active = t.seats[action]
	}
}

func (t *Table) payout() {
	hands := map[*Player]*hand.Hand{}
	for _, seat := range t.seats {
		hands[seat] = hand.New(append(seat.Cards, t.cards...))
	}
	for _, pot := range t.pots() {
		// sort by best hand first
		sort.Slice(pot.contesting, func(i, j int) bool {
			iHand := hands[pot.contesting[i]]
			jHand := hands[pot.contesting[j]]
			return iHand.CompareTo(jHand) > 0
		})
		// select winners who split pot if more than one
		winners := []*Player{}
		h1 := hands[pot.contesting[0]]
		for _, seat := range pot.contesting {
			h2 := hands[seat]
			if h1.CompareTo(h2) != 0 {
				break
			}
			winners = append(winners, seat)
		}
		// sort closest to the button for spare chips in split pot
		sort.Slice(winners, func(i, j int) bool {
			iDist := t.distanceFromButton(winners[i])
			jDist := t.distanceFromButton(winners[j])
			return iDist < jDist
		})
		// payout chips
		for i, seat := range winners {
			seat.Chips += pot.chips / len(winners)
			if (pot.chips % len(winners)) > i {
				seat.Chips++
			}
		}
	}
}

type sidePot struct {
	contesting []*Player
	chips      int
}

func (t *Table) pots() []*sidePot {
	contesting := t.contesting()
	sort.Slice(contesting, func(i, j int) bool {
		return contesting[i].ChipsInPot < contesting[j].ChipsInPot
	})
	var costs []int
	for _, seat := range contesting {
		if contains(costs, seat.ChipsInPot) == false {
			costs = append(costs, seat.ChipsInPot)
		}
	}
	var pots []*sidePot
	for i, cost := range costs {
		pot := &sidePot{}
		min := 0
		if i != 0 {
			min = costs[i-1]
		}
		for _, seat := range t.seats {
			pot.chips += max(seat.ChipsInPot-min, 0)
		}
		for _, seat := range contesting {
			if seat.ChipsInPot >= cost {
				pot.contesting = append(pot.contesting, seat)
			}
		}
		pots = append(pots, pot)
	}
	return pots
}

// chipsInPots returns the total chips in play
func (t *Table) chipsInPots() int {
	var chips int
	for _, pot := range t.pots() {
		chips += pot.chips
	}
	return chips
}

func (t *Table) resetAction() {
	for _, seat := range t.seats {
		if seat != nil {
			seat.Acted = false
		}
	}
}

func (t *Table) nextSeat(seat int) int {
	for {
		seat = (seat + 1) % len(t.seats)
		p := t.seats[seat]
		if p != nil {
			return seat
		}
	}
}

func (t *Table) nextToAct() int {
	count := 0
	seat := t.active.Seat
	for {
		seat = t.nextSeat(seat)
		p := t.seats[seat]
		if !p.Acted && !p.AllIn && !p.Folded {
			return p.Seat
		}
		count++
		if count == t.occupiedSeats()-1 {
			return -1
		}
	}
}

func (t *Table) occupiedSeats() int {
	count := 0
	for _, seat := range t.seats {
		if seat != nil {
			count++
		}
	}
	return count
}

func (t *Table) owed() int {
	return t.cost - t.active.ChipsInPot
}

func (t *Table) distanceFromButton(p *Player) int {
	seat := t.button
	dist := 0
	for {
		seat = t.nextSeat(seat)
		dist++
		if p.Seat == seat {
			return dist
		}
	}
}

func (t *Table) contesting() []*Player {
	var contesting []*Player
	for _, seat := range t.seats {
		if seat != nil && seat.Folded == false {
			contesting = append(contesting, seat)
		}
	}
	return contesting
}

type Player struct {
	ID         string      `json:"id"`
	Seat       int         `json:"seat"`
	Chips      int         `json:"chips"`
	ChipsInPot int         `json:"chipsInPot"`
	Acted      bool        `json:"acted"`
	Folded     bool        `json:"folded"`
	AllIn      bool        `json:"allIn"`
	Cards      []hand.Card `json:"cards"`
}

func (p *Player) contribute(chips int) {
	if p.Chips <= chips {
		chips = p.Chips
		p.AllIn = true
	}
	p.ChipsInPot += chips
	p.Chips -= chips
}

func includes(actions []ActionType, include ...ActionType) bool {
	for _, a1 := range include {
		found := false
		for _, a2 := range actions {
			found = found || a1 == a2
		}
		if !found {
			return false
		}
	}
	return true
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

func contains(a []int, i int) bool {
	for _, v := range a {
		if v == i {
			return true
		}
	}
	return false
}
