package main

//import "github.com/rightly/whoami-go/server"
import "whoami/server"



func main() {
	serv := server.New()
	serv.Start()
}