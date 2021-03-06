package table_test

import (
	"math/rand"
	"testing"

	"github.com/kolanos/goker/hand"
	"github.com/kolanos/goker/table"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	start       *table.Table
	actions     []table.Action
	condition   func(*table.State) bool
	description string
}

var (
	testCases = []testCase{
		{
			start:   threePerson100Buyin(),
			actions: nil,
			condition: func(s *table.State) bool {
				return s.Seats[0].Chips == 98 && s.Seats[1].Chips == 100 && s.Seats[2].Chips == 99 && s.Active.Seat == 1
			},
			description: "initial blinds",
		},
		{
			start: threePerson100Buyin(),
			actions: []table.Action{
				{table.Raise, 5},
			},
			condition: func(s *table.State) bool {
				return s.Seats[0].Chips == 98 && s.Seats[1].Chips == 93 && s.Seats[2].Chips == 99 && s.Active.Seat == 2 && s.Cost == 7
			},
			description: "preflop raise",
		},
		{
			start: threePerson100Buyin(),
			actions: []table.Action{
				{table.Raise, 5},
				{table.Call, 0},
				{table.Fold, 0},
				{table.Check, 0},
				{table.Bet, 5},
				{table.Fold, 0},
			},
			condition: func(s *table.State) bool {
				return s.Seats[0].Chips == 97 && s.Seats[1].Chips == 107 && s.Seats[2].Chips == 93 && s.Active.Seat == 2 && s.Button == 2
			},
			description: "full hand 1",
		},
	}
)

func TestTable(t *testing.T) {
	for _, tc := range testCases {
		tbl := tc.start

		for _, a := range tc.actions {
			assert.Nil(t, tbl.Act(a))
		}
		assert.Truef(t, tc.condition(tbl.State()), tc.description)
	}
}

func threePerson100Buyin() *table.Table {
	src := rand.NewSource(42)
	r := rand.New(src)
	dealer := hand.NewDealer(r)
	opts := table.Options{
		Variant: table.TexasHoldem,
		Limit:   table.NoLimit,
		Stakes:  table.Stakes{SmallBlind: 1, BigBlind: 2},
		BuyIn:   100,
	}
	ids := []string{"a", "b", "c"}
	return table.New(dealer, opts, len(ids), ids)
}

func TestTableNew(t *testing.T) {
	tbl := threePerson100Buyin()
	seats := tbl.Seats()
	assert.Equal(t, 3, len(seats))
	for i, p := range seats {
		assert.NotNil(t, p)
		assert.Equal(t, i, p.Seat)
	}
}
