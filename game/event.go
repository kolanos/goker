package game

import (
	"encoding/json"
	"errors"
)

type EventType string

const (
	Register    EventType = "register"
	SendMessage EventType = "sendMessage"
	Sit         EventType = "sit"
	Stand       EventType = "stand"
	Fold        EventType = "fold"
	Check       EventType = "check"
	Call        EventType = "call"
	Bet         EventType = "bet"
	Error       EventType = "error"
)

type Event struct {
	Event EventType   `json:"event"`
	data  interface{} `json:"data"`
}

type RegisterData struct {
	Name string `json:"name"`
}

type SendMessageData struct {
	Message string `json:"message"`
}

type BetData struct {
	Amount int `json:"amount"`
}

func ParseEvent(b []byte) (*Event, error) {
	var event *Event
	if err := json.Unmarshal(b, event); err != nil {
		return nil, errors.New("Unrecognized event type")
	}

	switch event.Event {
	case Fold:
	case Check:
	case Call:
		return event, nil
	case Register:
		data, err := parseRegister(b)
		if err != nil {
			return nil, err
		}
		return &Event{event.Event, data}, nil
	}

	return nil, errors.New("Unknown event type")
}

func parseRegister(b []byte) (*RegisterData, error) {
	var data *RegisterData
	if err := json.Unmarhsal(b, data); err != nil {
		return nil, err
	}
	return data, nil
}

func parseSendMessage(b []byte) (*SendMessageData, error) {
	var data *SendMessageData
	if err := json.Unmarhsal(b, data); err != nil {
		return nil, err
	}
	return data, nil
}
