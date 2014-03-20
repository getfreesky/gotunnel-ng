package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// arguments
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s [server-listen_address] [local-server_address-socks_address]\n", os.Args[0])
		return
	}
	var serverSessionManagers []*SessionManager
	var localSessionManager *SessionManager
	serverDeliveries := make(map[int64]*Delivery)
	for _, arg := range os.Args[1:] {

		if strings.HasPrefix(arg, "server") { // start server
			parts := strings.Split(arg, "-")
			listener, err := NewDeliveryListener(parts[1])
			if err != nil {
				log.Fatal(err)
			}
			listener.OnSignal("notify:delivery", func(delivery *Delivery) {
				serverDeliveries[delivery.Source] = delivery
				delivery.OnClose(func() {
					delete(serverDeliveries, delivery.Source)
				})
				m := NewSessionManager(delivery)
				serverSessionManagers = append(serverSessionManagers, m)
			})

		} else if strings.HasPrefix(arg, "local") { // start local
			parts := strings.Split(arg, "-")
			delivery, err := NewOutgoingDelivery(parts[1])
			if err != nil {
				log.Fatal(err)
			}
			localSessionManager = NewSessionManager(delivery)
			socksServer, err := NewSocksServer(parts[2])
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

		} else if strings.HasPrefix(arg, "status") { // status
			parts := strings.Split(arg, "-")
			go func() {
			err := http.ListenAndServe(parts[1], nil)
			if err != nil {
				log.Fatal(err)
			}
			}()
			// web status
			http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
				p := func(format string, args ...interface{}) {
					fmt.Fprintf(w, format, args...)
				}
				// status
				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)
				p("%d goroutines, %fm in use\n", runtime.NumGoroutine(), float64(memStats.Alloc)/1000000)
				p("load")
				for _, delivery := range serverDeliveries {
					p(" %f", delivery.Load)
				}
				p("\n")
				p("\n")
				// local sessions
				if localSessionManager != nil {
					p("%d local sessions\n", len(localSessionManager.Sessions))
					for _, session := range localSessionManager.Sessions {
						p("%v %v %s\n", session.localClosed, session.remoteClosed, session.hostPort)
					}
					p("\n")
				}
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
				var traces []string
				for trace, _ := range stats {
					traces = append(traces, trace)
				}
				sort.Strings(traces)
				for _, trace := range traces {
					p("%d\n", stats[trace])
					p("%s\n", trace)
				}
			})
		} else {
			log.Fatalf("unknown argument %s", arg)
		}
	}

	<-(make(chan bool))
}
