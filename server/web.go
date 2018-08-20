package server

import (
	"net/http"
	"strings"
	"time"
	"encoding/json"
	"fmt"
)

func NewHttpServer() *Http {
	s := &Http{
		Server: &http.Server{},
		Mux: http.NewServeMux(),
	}
	s.Server.Addr = ":80"

	return s
}

func (s *Server) ListenAndServe() error {
	// Set Handlers
	mux := s.Api.Mux
	mux.HandleFunc("/collect", s.webDiag())
	mux.HandleFunc("/show", func(res http.ResponseWriter, req *http.Request) {
		q, ok := req.URL.Query()["id"]
		res.WriteHeader(http.StatusOK)

		if !ok || len(q[0]) < 1 {
			body, err := json.MarshalIndent(s.Client, "", "\t")
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			res.Write(body)
			return
		}
		id := q[0]
		body, err := json.MarshalIndent(s.Client[id], "", "\t")
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.Write(body)
	})
	s.Api.Server.Handler = mux

	err := s.Api.Server.ListenAndServe()

	return err
}

func (s *Server) webDiag() func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		ip := strings.Split(req.RemoteAddr, ":")[0]
		ua := req.UserAgent()

		reqId := s.receive(ip, ua)

		res.WriteHeader(http.StatusOK)
		info, _ := json.Marshal(map[string]interface{}{
			"requestId": reqId,
			"info": s.Client[reqId],
		})
		fmt.Println(string(info))
		res.Write(info)
	}
}

func (s *Server) receive(ip, ua string) string {

	// request id receive
	reqID := <- s.RequestId
	// request id 에 client ip, ua, time 추가
	reqtime := time.Now().Format("2006-01-02 15:04:05 MST")
	s.Client[reqID].UserAgent = ua
	s.Client[reqID].Ip = ip
	s.Client[reqID].RequestTime = reqtime

	return reqID
}