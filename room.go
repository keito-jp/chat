package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/keito-jp/chat/trace"
	"github.com/stretchr/objx"
	"log"
	"net/http"
)

type room struct {
	// forward chan *message
	join    chan *client
	leave   chan *client
	clients map[*client]bool
	tracer  trace.Tracer
	psc redis.PubSubConn
}

func newRoom(c redis.Conn) *room {
	return &room{
		// forward: make(chan *message),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:  trace.Off(),
		psc: redis.PubSubConn{Conn: c},
	}
}

func (r *room) subscribe(channel string) {
	if r.psc.Conn == nil {
		log.Println("subscribeできません。psc.Connがセットされていません。")
		return
	}
	r.psc.Subscribe(channel)
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			r.clients[client] = true
			r.tracer.Trace("新しいクライアントが参加しました")
		case client := <-r.leave:
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("クライアントが退室しました")
		// case msg := <-r.forward:
		}
	}
}

func (r *room) receive() {

	for {
		switch n := r.psc.Receive().(type) {
		case redis.Message:
			msg := decodeJson(n.Data)
			// log.Println("Message: ", n.Channel, msg)
			r.tracer.Trace("メッセージを受信しました: ", msg.Message)
			for client := range r.clients {
				select {
				case client.send <- msg:
					// メッセージを送信
					r.tracer.Trace(" -- クライアントに送信されました")
				default:
					// 送信に失敗
					delete(r.clients, client)
					close(client.send)
					r.tracer.Trace(" -- 送信に失敗しました。クライアントをクリーンアップします")
				}
			}
		case redis.Subscription:
			log.Println("Subscription: ", n.Kind, n.Channel, n.Count)
		case error:
			log.Println("error: ", n)
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}

	authCookie, err := req.Cookie("auth")
	if err != nil {
		log.Fatal("クッキーの取得に失敗しました:", err)
		return
	}
	client := &client{
		socket:   socket,
		send:     make(chan *message, messageBufferSize),
		room:     r,
		userData: objx.MustFromBase64(authCookie.Value),
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
