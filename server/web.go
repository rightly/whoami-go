package server

import (
	"net/http"
	"strings"
	"time"
	"encoding/json"
	"fmt"
	"log"
)

func NewHttpServer() *Http {
	s := &Http{
		Server: &http.Server{},
		Mux: http.NewServeMux(),
	}
	s.Server.Addr = ":"+webPort

	return s
}

func (s *Server) ListenAndServe() error {
	// Set Handlers
	mux := s.Api.Mux
	mux.HandleFunc("/collect", Logger(s.webDiag))
	mux.HandleFunc("/show", BasicAuth(Logger(s.show)))
	s.Api.Server.Handler = mux

	err := s.Api.Server.ListenAndServe()

	return err
}

func (s *Server) webDiag(res http.ResponseWriter, req *http.Request) {
	ip := strings.Split(req.RemoteAddr, ":")[0]
	ua := req.UserAgent()

	reqId := s.receive(ip, ua)
	// timeout 일때 -> dns caching 으로 인해 룩업하지 않은 경우
	if reqId == "" {
		res.WriteHeader(http.StatusOK)
		body := "Your browser is already resolving and caching this domain"
		res.Write([]byte(body))
		return
	}

	res.WriteHeader(http.StatusOK)
	info, _ := json.Marshal(map[string]interface{}{
		"requestId": reqId,
		"info":      s.Client[reqId],
	})
	res.Write(info)
	return
}

func (s *Server)show(res http.ResponseWriter, req *http.Request) {
	// show 는 측정을 위한 요청이 아님으로 생성된 request id 제거
	s.delete()
	reqid := queryString(req, "id")

	// 전체 조회
	if reqid == "" {
		body, err := json.MarshalIndent(s.Client, "", "\t")
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Write(body)
		return
	}

	// receive time 추가
	receivT := queryString(req, "t")
	if receivT != "" {
		s.Client[reqid].ReceiveTime = receivT
	}

	// 특정 request id 조회
	body, err := json.MarshalIndent(s.Client[reqid], "", "\t")
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write(body)
}

func (s *Server) receive(ip, ua string) string {
	defer s.mu.Unlock()
	var reqId string
	timeout := time.After(1 * time.Second)

	s.mu.Lock()
	select {
	case reqId = <-s.RequestId:
		reqtime := time.Now().Format("2006-01-02 15:04:05 MST")
		s.Client[reqId].UserAgent = ua
		s.Client[reqId].Ip = ip
		s.Client[reqId].ResponseTime = reqtime
		return reqId
	case <-timeout:
		// timeout 인 경우 현재 request id 채널이 비어있다고 간주함
		return ""
	}
}

func (s *Server) delete()  {
	defer s.mu.Unlock()
	var reqId string
	timeout := time.After(1 * time.Second)

	s.mu.Lock()
	select {
	case reqId = <- s.RequestId:
		delete(s.Client, reqId)
	case <-timeout:
		// timeout 인 경우 현재 request id 채널이 비어있다고 간주함
	}
}

func queryString(req *http.Request, key string) string {
	q, ok := req.URL.Query()[key]

	if !ok || len(q[0]) < 1 {
		return ""
	}
	fmt.Println(q)
	return q[0]
}

func BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		username, password, authOK := r.BasicAuth()
		if authOK == false {
			http.Error(w, "Not authorized", 401)
			return
		}

		if username != "username" || password != "password" {
			http.Error(w, "Not authorized", 401)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func Logger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[api]: %v %v %v", r.RequestURI, r.UserAgent(), w.Header())
		next.ServeHTTP(w, r)
	}
}