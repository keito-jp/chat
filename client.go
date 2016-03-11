package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/garyburd/redigo/redis"
	"time"
	// "log"
)

type client struct {
	socket *websocket.Conn
	send chan *message
	room *room
	userData map[string]interface{}
}

func publish(channel, value interface{}) {
	c, err := redis.Dial("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()
	c.Do("PUBLISH", channel, value)
}

func (c *client) read() {
	for {
		var msg *message
		if err := c.socket.ReadJSON(&msg); err == nil {
			msg.When = time.Now()
			msg.Name = c.userData["name"].(string)
			if avatarURL, ok := c.userData["avatar_url"]; ok {
				msg.AvatarURL = avatarURL.(string)
			}
			publish("room01", msg.encodeJson())
			// c.room.forward <- msg
		} else {
			break
		}
	}
	c.socket.Close()
}

func (c *client) write() {
	for msg := range c.send {
		if err := c.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	c.socket.Close()
}
