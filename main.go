package main

import (
	"fmt"
	"net"
	"net/http"
	"io"
)


func getPair(pair string) {
	link := fmt.Sprintf("https://api.kraken.com/0/public/Ticker?pair=%s",pair)
	resp, err := http.Get(link)
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
}

func handleConnection(conn net.Conn) {
	conn.RemoteAddr()
	defer conn.Close()
}

func listenServer(port uint) {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Erreur lors de l'écoute :", err)
		return
	}
	fmt.Println("🚀 Serveur en écoute sur le port", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erreur d'acceptation :", err)
			continue
		}
		go handleConnection(conn)
	}
}

func main() {
	listenServer(8080)
}
