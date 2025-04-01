package main

import (
	"fmt"
	"net"
)

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
