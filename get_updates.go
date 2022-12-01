package main

import "time"

type Message struct {
	From string
	Text string
	Time time.Time
}

type GetUpdatesReply struct {
	Messages []*Message
}
