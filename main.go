package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kolanos/goker/game"
	"gopkg.in/olahol/melody.v1"
)

var g *game.Game

func init() {
	g = &game.Game{}
}

func main() {
	r := gin.Default()
	m := melody.New()

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleConnect(func(s *melody.Session) {
		g.Players.Join(s)
	})

	m.HandleDisconnect(func(s *melody.Session) {
		g.Players.Leave(s)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})

	r.Run(":5000")
}
