package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	go http.ListenAndServe("localhost:55555", nil)
}

func main() {
	// server
	listener, err := NewDeliveryListener("localhost:35000")
	if err != nil {
		log.Fatal(err)
	}
	var serverSessionManagers []*SessionManager
	listener.OnSignal("notify:delivery", func(delivery *Delivery) {
		m := NewSessionManager(delivery)
		serverSessionManagers = append(serverSessionManagers, m)
	})
	// local
	delivery, err := NewOutgoingDelivery("localhost:35000")
	if err != nil {
		log.Fatal(err)
	}
	localSessionManager := NewSessionManager(delivery)
	socksServer, err := NewSocksServer("localhost:31080")
	if err != nil {
		log.Fatal(err)
	}
	socksServer.OnSignal("client", func(conn net.Conn, hostPort string) {
		session, err := NewOutgoingSession(delivery, hostPort, conn)
		if err != nil {
			log.Fatal(err)
		}
		localSessionManager.Sessions[session.id] = session
		session.OnClose(func() {
			delete(localSessionManager.Sessions, session.id)
		})
	})

	// web status
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		p := func(format string, args ...interface{}) {
			fmt.Fprintf(w, format, args...)
		}
		// status
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		p("%d goroutines, %fm in use\n", runtime.NumGoroutine(), float64(memStats.Alloc)/1000000)
		p("\n")
		// local sessions
		p("%d local sessions\n", len(localSessionManager.Sessions))
		for _, session := range localSessionManager.Sessions {
			p("%v %v %s\n", session.localClosed, session.remoteClosed, session.hostPort)
		}
		p("\n")
		// server sessions
		for _, manager := range serverSessionManagers {
			p("%d server sessions\n", len(manager.Sessions))
			for _, session := range manager.Sessions {
				p("%v %v %s\n", session.localClosed, session.remoteClosed, session.hostPort)
			}
			p("\n")
		}
		// goroutines
		p("stack traces\n")
		records := make([]runtime.StackRecord, runtime.NumGoroutine()*2)
		n, _ := runtime.GoroutineProfile(records)
		stats := make(map[string]int)
		for i := 0; i < n; i++ {
			stack := records[i].Stack()
			entries := make([]string, 0)
			for _, pc := range stack {
				f := runtime.FuncForPC(pc)
				file, line := f.FileLine(pc)
				if !strings.Contains(file, "gotunnel-ng") {
					continue
				}
				entries = append(entries, fmt.Sprintf("%s %d", file, line))
			}
			if len(entries) > 0 {
				s := strings.Join(entries, "\n")
				stats[s]++
			}
		}
		for trace, count := range stats {
			p("%d\n", count)
			p("%s\n", trace)
		}
	})

	<-(make(chan bool))
}
