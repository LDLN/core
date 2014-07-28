package main

import (
	"github.com/ldln/landline-basestation/ws/chat"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// websocket server
	server := chat.NewServer("/ws")
	go server.Listen()

	// static files
	http.Handle("/", http.FileServer(http.Dir("webroot")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
