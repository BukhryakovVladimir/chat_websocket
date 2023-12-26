package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type room struct {
	cacheFilePath string

	messageCache [][]byte

	// clients holds all current clients in this room.
	clients map[*client]struct{}

	// join is a channel for clients wishing to join the room.
	join chan *client

	// leave is a channel for clients wishing to leave the room.
	leave chan *client

	// forward is a channel that holds incoming messages that should be forwarded to the other clients.
	forward chan []byte
}

// newRoom create a new chat room

func newRoom() *room {
	return &room{
		cacheFilePath: "cache.txt",
		messageCache:  make([][]byte, 0),
		forward:       make(chan []byte),
		join:          make(chan *client),
		leave:         make(chan *client),
		clients:       make(map[*client]struct{}),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = struct{}{}
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.receive)
		case msg := <-r.forward:
			for client := range r.clients {
				client.receive <- msg
			}
			if len(r.messageCache) < 10 {
				r.messageCache = append(r.messageCache, msg)
			} else {
				r.messageCache = append(r.messageCache[1:], msg)
			}
			WriteToFile(r.cacheFilePath, r.messageCache)
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket:  socket,
		receive: make(chan []byte, messageBufferSize),
		room:    r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}

func WriteToFile(filePath string, messageCache [][]byte) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, message := range messageCache {
		_, err := writer.Write(message)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		_, err = writer.WriteString("\n")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	err = writer.Flush()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}
