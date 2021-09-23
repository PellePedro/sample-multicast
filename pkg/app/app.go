package app

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"pellep.io/pkg/pwospf"
	"pellep.io/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	helloIntervalEnv = "HELLO_INTERVAL_MS" // Environment for interval to send OSPF Hello
)

var (
	once          sync.Once
	helloInterval = 5000
)

type Router struct {
	RouterID  uint32
	inCh      chan interface{}
	outCh     chan interface{}
	closureCh chan func()
}

func NewRouter(inChannel, outChannel chan interface{}) *Router {
	return &Router{
		inCh:      inChannel,
		outCh:     outChannel,
		closureCh: make(chan func(), 100),
	}
}

func init() {
	if interval, ok := os.LookupEnv(helloIntervalEnv); ok {
		var err error
		helloInterval, err = strconv.Atoi(interval)
		if err != nil {
			panicStr := fmt.Sprintf("Failed to parse [%s] value[%#v]", helloIntervalEnv, interval)
			panic(panicStr)
		}
	}

}

func (r *Router) Start() {
	for {
		select {

		case data := <-r.inCh:
			switch msg := data.(type) {
			case pwospf.NicInfo:
				once.Do(func() {
					r.RouterID = msg.IP
					go r.startHelloTimers()
				})

			case pwospf.PWOSPF:
				switch msg.Content.(type) {
				case pwospf.PWOspfLsu:
					r.handleLinkStateUpdate(msg)
				case pwospf.PWOspfHello:
					r.handleHello(msg)
				default:
					log.Error("Unexpected OSPF Type")
				}
			}

		case closure := <-r.closureCh:
			closure()
		}
	}
}

func (r *Router) startHelloTimers() {
	tick := time.NewTicker(time.Duration(helloInterval) * time.Millisecond)
	for range tick.C {
		r.SendHello()
	}
}

func (r *Router) SendHello() {
	log.Debugf("Send Hello from router [%s]", utils.IPUint32toStr(r.RouterID))
	builder := pwospf.NewHello()
	builder.SetRouterID(r.RouterID)
	hello := builder.BuildRequest()
	r.outCh <- hello
}

func (r *Router) handleHello(req pwospf.PWOSPF) {
	log.Infof("received Hello from router [%s], link [%s] ",
		utils.IPUint32toStr(req.RouterID), req.LinkName)
}

func (r *Router) sendLinkStateUpdate(req pwospf.PWOSPF) {
	log.Debug("sendLinkStateUpdate")
}

func (r *Router) handleLinkStateUpdate(req pwospf.PWOSPF) {
	log.Debug("handleLinkStateUpdate")
}
