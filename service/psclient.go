package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"surrealchemist.com/mass-showdown-backend/messages"
)

const SIM_URL = "wss://sim3.psim.us/showdown/websocket"
const SIM_ACTION_URL = "https://play.pokemonshowdown.com/~~showdown/action.php"
const AUTHORIZED_OPP = "hosergang"
const AUTHORIZED_FORMAT = "gen9randombattle"

type PSClient struct {
	username, pass string
	log            *zap.SugaredLogger
	inbox          chan *message
	outbox         chan *message
	wg             *sync.WaitGroup
	inBattle       bool
}

func NewPSClient(wg *sync.WaitGroup) *PSClient {
	return &PSClient{
		username: "cruisergang",
		pass:     "groovedune",
		log:      zap.NewExample().Sugar().Named("ps_client"),
		inbox:    make(chan *message),
		wg:       wg,
		inBattle: false,
	}
}

func (p *PSClient) GetRecvChan() chan *message {
	return p.inbox
}

func (p *PSClient) SetSendChan(send chan *message) {
	p.outbox = send
}

func (p *PSClient) LoginAndStart() {
	defer p.wg.Done()
	if p.outbox == nil {
		p.log.Fatalw("showdown client currently has no channel set for communicating with poll server")
	}
	c, _, err := websocket.DefaultDialer.Dial(SIM_URL, nil)
	if err != nil {
		p.log.Fatalw("Error opening Websocket", zap.Error(err))
	}
	defer c.Close()

	go func() {
		for {
			_, bs, err := c.ReadMessage()
			if err != nil {
				p.log.Fatalw("Error reading websocket message", zap.Error(err))
			}
			p.handleWS(bs, c)
		}
	}()

	for msg := range p.inbox {
		switch msg.Type {
		case results:
			content, ok := msg.Content.(pollResults)
			if !ok {
				p.log.Warn("Got pollResults message from poll server with unrecognized payload")
				break
			}
			wsm := fmt.Sprintf("%s|%s|%d", content.RoomID, content.Command, content.RQID)
			p.log.Infow("sending message to server",
				zap.String("content", wsm))
			c.WriteMessage(websocket.TextMessage, []byte(wsm))
		}
	}
}

func (p *PSClient) handleWS(bs []byte, c *websocket.Conn) {
	msg, err := messages.ParseServerMessage(bs)
	if err != nil {
		p.log.Fatalw("Error parsing websocket message", zap.Error(err))
	}
	for _, m := range msg.Messages {
		p.log.Infow("Received websocket message from server",
			zap.String("room", msg.RoomID),
			zap.String("type", m.Type),
			zap.Strings("data", m.Data),
			zap.Int("length", len(m.Data)),
		)
		switch m.Type {
		case "challstr":
			r, err := p.login(m.Data)
			if err != nil {
				p.log.Fatalw("Error logging in", zap.Error(err))
			}
			c.WriteJSON([]string{fmt.Sprintf("|/trn %s,0,%s", p.username, r.Assertion)})
			if err != nil {
				p.log.Fatalw("Error logging in", zap.Error(err))
			}
		case "pm":
			if !strings.HasPrefix(m.Data[2], "/challenge") {
				break
			}
			if m.Data[0][1:] != AUTHORIZED_OPP || m.Data[2] != "/challenge "+AUTHORIZED_FORMAT || p.inBattle {
				c.WriteJSON([]string{fmt.Sprintf("|/reject%s", m.Data[0])})
				break
			}
			c.WriteJSON([]string{fmt.Sprintf("|/accept%s", m.Data[0])})
			p.inBattle = true
		case "|request":
			// each battle starts with a blank request which needs to be ignored
			if m.Data[0] == "" {
				break
			}
			wsm := fmt.Sprintf("|/join %s", msg.RoomID)
			p.log.Infow("sending message", zap.String("content", wsm))
			c.WriteJSON([]string{})
			req := &PSBattleRequest{}
			err = json.Unmarshal([]byte(m.Data[0]), req)
			if err != nil {
				p.log.Errorf("couldn't unmarshal showdown json", zap.Error(err))
				break
			}
			p.outbox <- &message{
				Type: showdownRequest,
				Content: showdownRequestMessage{
					RoomID: msg.RoomID,
					Req:    req,
				},
			}
		case "win":
			wsm := fmt.Sprintf("|/leave %s", msg.RoomID)
			p.log.Infow("sending message", zap.String("content", wsm))
			c.WriteMessage(websocket.TextMessage, []byte(wsm))
			p.inBattle = false
			// resp := <-p.inbox
			// if resp.Type == wait {
			// 	break
			// }
			// wsm = fmt.Sprintf("%s|%s|%d", msg.RoomID, resp.Content, req.RQID)
			// p.log.Infow("sending message to server",
			// 	zap.String("content", wsm))
			// c.WriteMessage(websocket.TextMessage, []byte(wsm))
			// c.WriteJSON([]string{wsm})
		}
	}
}

type loginResponse struct {
	ActionSuccess bool
	Assertion     string
	CurrentUser   map[string]interface{}
}

func (p *PSClient) login(challdata []string) (*loginResponse, error) {
	log := zap.NewExample().Sugar().Named("login")
	resp, err := http.PostForm(SIM_ACTION_URL, url.Values{
		"act":      {"login"},
		"name":     {p.username},
		"pass":     {p.pass},
		"challstr": {strings.Join(challdata, "|")},
	})
	if err != nil {
		return nil, err
	}
	lr := &loginResponse{}
	respBytes, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(respBytes[1:], lr)
	if err != nil {
		return nil, err
	}
	if !lr.ActionSuccess {
		return nil, errors.New("failed logging in")
	}
	log.Infow("Got response from server on login", zap.ByteString("response", respBytes))
	return lr, err
}
