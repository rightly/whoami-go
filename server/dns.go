package server

import (
	"github.com/miekg/dns"
	"fmt"
	"os"
	"net"
)

// DNS Server Handle Functions
func (s *Server) DnsHandler() {
	dns.HandleFunc("whoami.hlight.tk.", whoami)
	dns.HandleFunc("diag.hlight.tk.", s.dnsDiag)
}

// whoami 는 client( local cache dns ) ip 를 return 해줌
func whoami(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4  bool
		rr  dns.RR
		str string
		a   net.IP
	)
	m := new(dns.Msg)
	m.SetReply(r)

	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
		v4 = a.To4() != nil
	}

	// Query 응답
	// Client Cache DNS IP에 대해 TTL 0
	// 단순 A, TXT 레코드 질의만 처리하기 때문에, Secure 또는 기타 옵션에 대한 예외처리 안함

	// IPv4 or IPv6
	if v4 {
		rr = &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   a.To4(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr:  dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0},
			AAAA: a,
		}
	}

	t := &dns.TXT{
		Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0},
		Txt: []string{str},
	}

	// TXT or A (AAAA)
	// TXT 레코드 질의일때는 Answer Section 에 Request ID 를
	// A (AAAA) 레코드 일즤일때는 Answer Section 에 Client Local cache server ip 를
	// 나버민 Additional Section 에 담아 응답
	switch r.Question[0].Qtype {
	case dns.TypeTXT:
		m.Answer = append(m.Answer, t)
		m.Extra = append(m.Extra, rr)
	default:
		fallthrough
	case dns.TypeAAAA, dns.TypeA:
		m.Answer = append(m.Answer, rr)
		m.Extra = append(m.Extra, t)
	}

	fmt.Fprintln(os.Stdout, m.String())

	w.WriteMsg(m)
}

// dnsDiag 는 diag web server ip 를 리턴해주며 requestId 를 생성해
// diag web server 에 전달한다
func (s *Server) dnsDiag(w dns.ResponseWriter, r *dns.Msg)  {
	var (
		a net.IP
		collector = "collect.hlight.tk"
	)

	m := new(dns.Msg)
	m.SetReply(r)


	ipArr, err := net.LookupIP(collector)
	if err != nil {
		fmt.Println("Can't resolve :", collector, " : ",err)
	}

	for _, ip := range ipArr {
		// IPv4 or IPv6
		rr := &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
			A:   ip.To4(),
		}
		m.Answer = append(m.Answer, rr)
	}

	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
	}

	w.WriteMsg(m)

	s.throw(&a)
	fmt.Fprintln(os.Stdout, m.String())
}

func (s *Server) throw(ldns *net.IP)  {
	go func() {
		// Random request id 생성
		reqId := random.String(32)
		// request id 에 local cache dns ip 추가
		s.Client[reqId] = &Info{
			Dns:*ldns,
		}
		// request id 전달
		s.RequestId <- reqId
	}()
}