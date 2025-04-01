package main

import (
	"fmt"
	"net"
	"net/http"
	"io"
	"encoding/json"
	"log"
)

type Result struct {
	Error []string `json:"error"`
	Result struct {
	}`json:"result"`
}

type ServerTime struct {
	unixtime int `json:"unixtime"`
	rfc1123 string `json:"rfc1123"`
}

type ServerSystemStatus struct {
	status string `json:"status"`
}

type Tickers struct {
	Pair []struct{
		v []string `json:"v"`
		h []string `json:"h"`
		l []string `json:"l"`
	}
}

func getPair(pair string) {
	link := fmt.Sprintf("https://api.kraken.com/0/public/Ticker?pair=%s",pair)
	resp, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	return body
}

func getAllPair() []byte{
	resp, err := http.Get("https://api.kraken.com/0/public/Ticker")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	return body
}

func parseResponse( response []byte) Result {
	var value Result
	err := json.Unmarshal([]byte(body),&value)
	if err != nil {
		log.Fatal(err)
	}
	return value
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
	pairs := getAllPair()
	fmt.Println(parseResponse(pairs))
	listenServer(8080)
}
