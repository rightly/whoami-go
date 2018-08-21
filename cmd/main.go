package main

import "github.com/rightly/whoami-go/server"



func main() {
	serv := server.New()
	serv.Start()
}