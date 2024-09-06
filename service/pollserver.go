package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

var AUTHORIZED_HOSTS = [...]string{"localhost:8080"}

type PollServer struct {
	upgrader     *websocket.Upgrader
	serverInbox  chan *message
	serverOutbox chan *message
	wg           *sync.WaitGroup
	pool         *pollWorkerPool
	log          *zap.SugaredLogger
}

type Poll struct {
	Req            *PSBattleRequest
	RoomID         string
	StartedAt      time.Time
	EndsAt         time.Time
	Attack, Switch []int16
	Tera           uint16
	Total          uint16
}

type pollWorker struct {
	id    string
	voted bool
	inbox chan *message
}

type pollWorkerPool struct {
	sync.Mutex
	managerInbox chan *message
	workers      map[string]*pollWorker
}

func NewPollServer(wg *sync.WaitGroup) *PollServer {
	u := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	u.CheckOrigin = checkOrigin
	return &PollServer{
		upgrader:    u,
		serverInbox: make(chan *message),
		pool:        initPollWorkerPool(),
		wg:          wg,
		log:         zap.NewExample().Sugar().Named("pollserver"),
	}
}

func checkOrigin(r *http.Request) bool {
	for _, host := range AUTHORIZED_HOSTS {
		if r.Host == host {
			return true
		}
	}
	return false
}

// Starts the server, then listens for messages as the manager.
// Essentially, this is the manager loop.
func (p *PollServer) StartServer() {
	defer p.wg.Done()
	http.HandleFunc("/ws", p.wsServerHandler)
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./service/static"))))
	go http.ListenAndServe(":8080", nil)
	var po *Poll
	for {
		select {
		case msg := <-p.serverInbox:
			switch msg.Type {
			case showdownRequest:
				req, ok := msg.Content.(showdownRequestMessage)
				if !ok {
					p.log.Errorw("received request with unexpected payload",
						zap.String("type", string(msg.Type)),
						zap.Any("content", msg.Content))
					break
				}
				if req.Req.Wait {
					p.serverOutbox <- &message{
						Type:    wait,
						Content: nil,
					}
					break
				}
				po = &Poll{
					Req:       req.Req,
					RoomID:    req.RoomID,
					StartedAt: time.Now(),
					EndsAt:    time.Now().Add(30 * time.Second),
					Attack:    make([]int16, 4),
					Switch:    make([]int16, 6),
				}
				p.pool.Broadcast(&message{
					Type: updateResponse,
					Content: updateResponseMessage{
						Results: false,
						Update:  req,
					},
				})
				p.log.Infow("started poll", zap.Any("poll", po))
			}
		case msg := <-p.pool.managerInbox:
			switch msg.Type {
			case vote:
				if po == nil {
					p.log.Warnw("received SendVote message but no poll was active")
					break
				}
				v, ok := msg.Content.(*Vote)
				if !ok {
					p.log.Warnw("received SendVote message but data was not a vote", zap.Any("data", msg.Content))
					break
				}
				if v.Type == "move" {
					if len(po.Req.ForceSwitch) > 0 && po.Req.ForceSwitch[0] {
						p.log.Warnw("received vote to move when poll should force switch",
							zap.String("id", v.From))
						p.pool.SendToWorker(v.From, &message{
							Type: clearVote,
						})
						break
					}
					if v.Idx > len(po.Attack) || v.Idx < 0 {
						p.log.Warnw("received vote with index out of bounds",
							zap.String("id", v.From))
						p.pool.SendToWorker(v.From, &message{
							Type: clearVote,
						})
						p.pool.SendToWorker(v.From, &message{
							Type: displayText,
							Content: displayTextMessage{
								Clear:   false,
								Err:     true,
								Message: "Invalid selection",
							},
						})
						break
					}
					if po.Req.Active[0].Moves[v.Idx].Disabled {
						p.log.Warnw("received vote for disabled move",
							zap.String("id", v.From))
						p.pool.SendToWorker(v.From, &message{
							Type: clearVote,
						})
						p.pool.SendToWorker(v.From, &message{
							Type: displayText,
							Content: displayTextMessage{
								Clear:   false,
								Err:     true,
								Message: "Invalid selection",
							},
						})
						break
					}
					po.Attack[v.Idx]++
				} else {
					if v.Idx > len(po.Switch) || v.Idx < 0 {
						p.log.Warnw("received vote with index out of bounds",
							zap.String("id", v.From))
						p.pool.SendToWorker(v.From, &message{
							Type: clearVote,
						})
						p.pool.SendToWorker(v.From, &message{
							Type: displayText,
							Content: displayTextMessage{
								Clear:   false,
								Err:     true,
								Message: "Invalid selection",
							},
						})
						break
					}
					if po.Req.Side.Pokemon[v.Idx].Condition == "0 fnt" {
						p.log.Warnw("received vote for fainted pokemon",
							zap.String("id", v.From))
						p.pool.SendToWorker(v.From, &message{
							Type: clearVote,
						})
						p.pool.SendToWorker(v.From, &message{
							Type: displayText,
							Content: displayTextMessage{
								Clear:   false,
								Err:     true,
								Message: "Invalid selection",
							},
						})
						break
					}
					po.Switch[v.Idx]++
				}
				if v.Tera {
					po.Tera++
				}
				po.Total++
				p.pool.SendToWorker(v.From, &message{
					Type: voteOk,
				})
			case updateRequest:
				c, ok := msg.Content.(updateRequestMessage)
				if !ok {
					p.log.Errorw("received request with unexpected payload",
						zap.String("type", string(msg.Type)),
						zap.Any("content", msg.Content))
					break
				}
				if po == nil {
					p.pool.SendToWorker(c.From, &message{
						Type: displayText,
						Content: displayTextMessage{
							Clear:   true,
							Err:     false,
							Message: "Please wait...",
						},
					})
				} else if !c.Voted {
					p.pool.SendToWorker(c.From, &message{
						Type: updateResponse,
						Content: updateResponseMessage{
							Results: false,
							Update:  po.Req,
						},
					})
				} else {
					if len(po.Req.Active) > 0 {
						for i, a := range po.Req.Active[0].Moves {
							a.Votes = float32(po.Attack[i]) / float32(po.Total)
						}
						po.Req.Active[0].TeraVotes = float32(po.Tera) / float32(po.Total)
					}
					for i, a := range po.Req.Side.Pokemon {
						a.Votes = float32(po.Switch[i]) / float32(po.Total)
					}
					p.pool.SendToWorker(c.From, &message{
						Type: updateResponse,
						Content: updateResponseMessage{
							Results: true,
							Update:  po.Req,
						},
					})
				}
			}
		default:
			if po == nil || time.Now().Before(po.EndsAt) {
				break
			}
			p.pool.Broadcast(&message{
				Type: displayText,
				Content: displayTextMessage{
					Clear:   true,
					Err:     false,
					Message: "Please wait...",
				},
			})
			winnerT := "move"
			winnerId := 0
			winnerCt := int16(-1)
			tera := ""

			if len(po.Req.ForceSwitch) == 0 && len(po.Req.Active) > 0 {
				for i, ct := range po.Attack {
					if ct > winnerCt && !po.Req.Active[0].Moves[i].Disabled {
						winnerId = i
						winnerCt = ct
					}
				}
			}
			for i, ct := range po.Switch {
				if ct > winnerCt && po.Req.Side.Pokemon[i].Condition != "0 fnt" {
					winnerT = "switch"
					winnerId = i
					winnerCt = ct
				}
			}
			if winnerT == "move" && po.Tera*2 > po.Total {
				tera = " terastallize"
			}
			p.serverOutbox <- &message{
				Type: results,
				Content: pollResults{
					RoomID:  po.RoomID,
					RQID:    po.Req.RQID,
					Command: fmt.Sprintf("/choose %s %d%s", winnerT, winnerId+1, tera),
				},
			}
			po = nil
			p.pool.Broadcast(&message{
				Type:    clearVote,
				Content: "",
			})
		}
	}
}

func (p *PollServer) GetRecvChan() chan *message {
	return p.serverInbox
}

func (p *PollServer) SetSendChan(send chan *message) {
	p.serverOutbox = send
}

// The websocket handler stores its information and sends/receives through a worker.
// Essentially, this is the poll worker loop.
func (p *PollServer) wsServerHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := p.upgrader.Upgrade(w, r, nil)
	worker := p.pool.NewWorker()
	errct := 0
	if err != nil {
		p.log.Error("error upgrading to websocket connection",
			zap.Error(err),
			zap.Any("request", r),
		)
		return
	}
	defer ws.Close()
	wsChan := make(chan interface{})
	go func() {
		for {
			t, bytes, err := ws.ReadMessage()
			if t == websocket.CloseMessage || websocket.IsUnexpectedCloseError(err) {
				close(wsChan)
				return
			}
			if err != nil {
				errct++
				wsChan <- err
			} else {
				errct = 0
				wsChan <- bytes
			}
		}
	}()

	p.pool.managerInbox <- &message{
		Type: updateRequest,
		Content: updateRequestMessage{
			From:  worker.id,
			Voted: false,
		},
	}
	for {
		select {
		case msg := <-worker.inbox:
			switch msg.Type {
			case voteOk:
				fallthrough
			case displayText:
				fallthrough
			case updateResponse:
				ws.WriteJSON(msg)
			case clearVote:
				worker.voted = false
			}
		case msg := <-wsChan:
			if msg == nil {
				p.log.Infow("terminating worker because websocket was closed",
					zap.String("worker_id", worker.id))
				p.pool.KillWorker(worker.id)
				return
			}
			if err, ok := msg.(error); ok {
				p.log.Errorw("error reading from websocket",
					zap.String("worker_id", worker.id),
					zap.Error(err))
				if errct > 5 {
					p.log.Errorw("terminating worker because there were too many errors",
						zap.String("worker_id", worker.id))
					p.pool.KillWorker(worker.id)
					return
				}
				break
			}
			bytes := msg.([]byte)
			p.log.Infow("received from worker websocket",
				zap.String("worker_id", worker.id),
				zap.ByteString("bytes", bytes))
			m := &message{}
			err = json.Unmarshal(bytes, m)
			if err != nil {
				p.log.Errorf("couldn't unmarshal message json", zap.Error(err))
				break
			}
			switch m.Type {
			case vote:
				if worker.voted {
					p.log.Infow("received a vote from a client who already voted",
						zap.String("worker_id", worker.id))
					break
				}
				content, ok := m.Content.(map[string]interface{})
				if !ok {
					p.log.Errorw("received request with unexpected payload (expected Vote)",
						zap.String("type", string(m.Type)),
						zap.Any("content", m.Content))
					break
				}
				v := &Vote{
					From: worker.id,
					Type: content["type"].(string),
					Idx:  int(content["idx"].(float64)),
					Tera: content["tera"].(bool),
				}
				p.log.Infow("voted", zap.Any("vote", v))
				p.pool.managerInbox <- &message{
					Type:    vote,
					Content: v,
				}
				worker.voted = true
			case updateRequest:
				p.pool.managerInbox <- &message{
					Type: updateRequest,
					Content: updateRequestMessage{
						From:  worker.id,
						Voted: worker.voted,
					},
				}
			}
		}
	}
}

// Initializes a poll worker pool.
func initPollWorkerPool() *pollWorkerPool {
	return &pollWorkerPool{
		managerInbox: make(chan *message, 40),
		workers:      make(map[string]*pollWorker, 40),
	}
}

// Creates a worker for the pool and returns the newly created worker.
func (wp *pollWorkerPool) NewWorker() *pollWorker {
	id := ksuid.New().String()
	w := &pollWorker{
		id:    id,
		inbox: make(chan *message, 3),
	}
	wp.Lock()
	wp.workers[id] = w
	wp.Unlock()
	return w
}

// Sends a message to all workers in the pool.
func (wp *pollWorkerPool) Broadcast(msg *message) {
	wp.Lock()
	for _, w := range wp.workers {
		w.inbox <- msg
	}
	wp.Unlock()
}

// Sends a message to the specified worker.
func (wp *pollWorkerPool) SendToWorker(id string, msg *message) {
	wp.Lock()
	wp.workers[id].inbox <- msg
	wp.Unlock()
}

// Deletes a single worker from the pool.
func (wp *pollWorkerPool) KillWorker(id string) {
	wp.Lock()
	delete(wp.workers, id)
	wp.Unlock()
}

// Closes all channels for all workers in the pool.
func (wp *pollWorkerPool) Shutdown(msg *message) {
	wp.Lock()
	for _, w := range wp.workers {
		close(w.inbox)
	}
	wp.Unlock()
}

func (wp *pollWorkerPool) WorkerHasVoted(id string) bool {
	var voted bool
	wp.Lock()
	voted = wp.workers[id].voted
	wp.Unlock()
	return voted
}
