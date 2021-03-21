package game

import (
	"github.com/kolanos/goker/table"
	"gopkg.in/olahol/melody.v1"
)

type Game struct {
	Players *Players
	Tables  []table.Table
}

func New() *Game {
	return &Game{}
}

type Players map[*melody.Session]*table.Player

func (p *Players) Join(s *melody.Session) {
	if !p.Exists(s) {
		(*p)[s] = &table.Player{}
	}
}

func (p *Players) Leave(s *melody.Session) {
	delete(*p, s)
}

func (p *Players) Exists(s *melody.Session) bool {
	_, ok := (*p)[s]
	return ok
}

func (p *Players) Len() int {
	return len(*p)
}
