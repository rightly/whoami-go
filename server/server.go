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
)

// DNS, Web Server Port
const (
	webPort="8080"
	dnsPort="8053"
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
		Api *Http
		Dns *dns.Server
		// RequestID 에 맞게 client ip, client cache server ip 를 넣어준다.
		// e.g ) examRequestId[server:10.204.0.1, client:10.84.36.34]
		Client map[string]*Info
		// Job queue
		RequestId chan string
	}
)

func New() *Server {
	queueCount := 1

	return &Server{
		Api: NewHttpServer(),
		// DNS 메시지 길이는 512byte 가 넘지 않고 Zone Transfer 요청도 없음으로 udp 만
		Dns:       &dns.Server{Addr: "[::]:"+dnsPort, Net: "udp4", TsigSecret: nil},
		Client:    make(map[string]*Info, 0),
		RequestId: make(chan string, queueCount),
	}
}

func (v *Info) String() string {
	str := fmt.Sprintf("ip=%v,ua=%v,dns=%v", v.Ip, v.UserAgent, v.Dns)
	return str
}

func (s *Server) Start() {
	var err error
	// DNS Server
	go func() {
		s.DnsHandler()
		err = s.Dns.ListenAndServe()
		if err != nil {
			fmt.Printf("Failed to setup the dns server: %s\n", err.Error())
		}
	}()

	// API Server
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			fmt.Printf("Failed to setup the web server: %s\n", err.Error())
		}
	}()
	if err != nil {
		fmt.Println("server started: ", "web(", s.Api.Server.Addr, "), dns(", s.Dns.Addr, ")")
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	catch := <-sig

	fmt.Printf("Signal (%s) received, stopping\n", catch)
}