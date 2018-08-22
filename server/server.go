package server

import (
	"github.com/rightly/whoami-go/util"
	"github.com/miekg/dns"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"net/http"
	"time"
	"log"
	"sync"
	"encoding/json"
)

// DNS, Web Server Port
const (
	webPort="80"
	dnsPort="53"
)

var random = util.New()

type (
	// Diagnosis 정보
	Info struct {
		Dns          net.IP `json:"clientDns"`
		Ip           string `json:"clientIp"`
		UserAgent    string `json:"userAgent"`
		ResponseTime string `json:"responseTime"`
		ReceiveTime  string `json:"receiveTime"`
	}
	// Web Server
	Http struct {
		Server *http.Server
		Mux  *http.ServeMux
	}

	// Web, DNS Server
	Server struct {
		mu     *sync.Mutex
		Api    *Http
		Dns    *dns.Server
		Client map[string]*Info
		// Job queue
		RequestId chan string
	}
)

func New() *Server {
	queueCount := 1

	return &Server{
		mu: new(sync.Mutex),
		Api: NewHttpServer(),
		// DNS 메시지 길이는 512byte 가 넘지 않고 Zone Transfer 요청도 없음으로 udp 만
		Dns:       &dns.Server{Addr: "[::]:"+dnsPort, Net: "udp4", TsigSecret: nil},
		Client:    make(map[string]*Info, 0),
		RequestId: make(chan string, queueCount),
	}
}

func (v *Info) String() string {
	bytes, _ := json.Marshal(v)
	str := string(bytes)
	return str
}

func (s *Server) Start() {
	// DNS Server
	var err error
	go func() {
		s.DnsHandler()
		err = s.Dns.ListenAndServe()
		if err != nil {
			fmt.Printf("Failed to setup the dns server: %s\n", err.Error())
		}
	}()

	// API Server
	go func() {
		err = s.ListenAndServe()
		if err != nil {
			fmt.Printf("Failed to setup the web server: %s\n", err.Error())
		}
	}()

	go func() {
		s.garbageCollector(60 * time.Second)
	}()

	fmt.Println("server started: ", "web(", s.Api.Server.Addr, "), dns(", s.Dns.Addr, ")")


	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	catch := <-sig

	fmt.Printf("Signal (%s) received, stopping\n", catch)
}

// 60초마다 비어있는 requestid 를 제거
func (s *Server)garbageCollector(t time.Duration) {
	for {
		s.mu.Lock()
		for id, info := range s.Client {
			if info.Ip == "" && info != nil {
				delete(s.Client, id)
				log.Println("[gc]: delete ", id)
			}
		}
		s.mu.Unlock()
		time.Sleep(t)
	}
}