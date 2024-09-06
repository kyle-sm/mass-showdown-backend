package main

import (
	"sync"

	"surrealchemist.com/mass-showdown-backend/service"
)

func main() {
	// log := zap.NewExample().Sugar().Named("main")
	wg := &sync.WaitGroup{}
	psc := service.NewPSClient(wg)
	ps := service.NewPollServer(wg)
	psc.SetSendChan(ps.GetRecvChan())
	ps.SetSendChan(psc.GetRecvChan())
	wg.Add(2)
	go psc.LoginAndStart()
	go ps.StartServer()
	wg.Wait()
}

