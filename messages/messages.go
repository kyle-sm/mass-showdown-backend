package messages

import (
	"fmt"
	"unicode/utf8"
)

type (
	Message struct {
		Type string
		Data []string
	}
	ServerMessage struct {
		RoomID   string
		Messages []Message
	}

	ClientMessage struct {
		RoomID, Type, Text string
		ResponseID         int
	}
)

func (cm *ClientMessage) Marshal() []byte {
	return []byte(fmt.Sprintf("%s|%s%s|%d", cm.RoomID, cm.Type, cm.Text, cm.ResponseID))
}

func ParseServerMessage(msg []byte) (*ServerMessage, error) {
	sm := new(ServerMessage)
	r, size := utf8.DecodeRune(msg)
	pos := size
	curMsg := new(Message)
	lastBreak := -1
	switch r {
	case '>':
		for r != '\n' {
			r, size = utf8.DecodeRune(msg[pos:])
			pos += size
		}
		sm.RoomID = string(msg[1 : pos-size])
		fallthrough
	case '|':
		lastBreak = pos
		pos++
		for pos < len(msg) {
			r, size = utf8.DecodeRune(msg[pos:])
			if r == '|' {
				if curMsg.Type == "" {
					curMsg.Type = string(msg[lastBreak:pos])
				} else {
					curMsg.Data = append(curMsg.Data, string(msg[lastBreak:pos]))
				}
				lastBreak = pos + 1
			} else if r == '\n' {
				if curMsg.Type == "" {
					curMsg.Type = string(msg[lastBreak:pos])
				} else {
					curMsg.Data = append(curMsg.Data, string(msg[lastBreak:pos]))
				}
				lastBreak = pos + 1
				sm.Messages = append(sm.Messages, *curMsg)
				curMsg = new(Message)
			}
			pos += size
		}
	default:
		sm.Messages = append(sm.Messages, Message{Type: "raw", Data: []string{string(msg)}})
	}
	// If the message didn't end in a newline, we have to make sure to account for the rest of it
	if r != '\n' {
		if curMsg.Type == "" {
			curMsg.Type = string(msg[lastBreak:])
		} else {
			curMsg.Data = append(curMsg.Data, string(msg[lastBreak:]))
		}
		sm.Messages = append(sm.Messages, *curMsg)
	}
	return sm, nil
}
