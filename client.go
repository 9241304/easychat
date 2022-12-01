package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/containerd/console"
)

var name string

func clearPrevLine() {
	fmt.Print("\033[A\033[2KT\r")
}

func clearCurLine() {
	fmt.Print("\033[2K\r")
}

func clearScreen() {
	fmt.Print("\033c")
}

func printLineAndPrompt(line string) {
	clearCurLine()
	println(line)
	fmt.Print("Say -> ")
}

func LaunchClient() {
	//use external lib to setup windows terminal
	//will not work in debug mode
	console.Current()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Enter your name: ")
	re := regexp.MustCompile(`^[A-Za-z0-9_]{3,}$`)
	for {
		scanner.Scan()
		text := scanner.Text()

		if re.MatchString(text) {
			name = text
			break
		}

		clearPrevLine()
		fmt.Printf("Try again: ")
	}

	clearScreen()

	go getUpdatesLoop()

	for {
		fmt.Print("Say -> ")
		scanner.Scan()
		text := scanner.Text()

		clearPrevLine()

		if len(text) != 0 {
			data := PostMessageRequest{
				From: name,
				Text: text,
			}
			js, _ := json.Marshal(&data)
			http.Post("http://localhost:14300/postMessage", "application/json", bytes.NewBuffer(js))
		} else {
			// exit if user entered an empty string
			break
		}
	}
}

func getUpdatesLoop() {
	for {
		printLineAndPrompt(fmt.Sprintf("[%v] Sent getUpdates request to server", time.Now().Format("15:04:05.000")))
		resp, err := http.Get(fmt.Sprintf("http://localhost:14300/getUpdates?for=%v", name))

		if err != nil {
			printLineAndPrompt(fmt.Sprintf("[%v] Received error on getUpdates from server", time.Now().Format("15:04:05.000")))
			time.Sleep(10 * time.Second)
		} else {
			var reply GetUpdatesReply
			dec := json.NewDecoder(resp.Body)
			dec.DisallowUnknownFields()
			err = dec.Decode(&reply)
			if err != nil {
				printLineAndPrompt(fmt.Sprintf("[%v] Received invalid getUpdates reply from server", time.Now().Format("15:04:05.000")))
			} else {
				printLineAndPrompt(fmt.Sprintf("[%v] Received correct getUpdates reply from server", time.Now().Format("15:04:05.000")))
				for _, msg := range reply.Messages {
					printLineAndPrompt(fmt.Sprintf("[%v]->[%v] %v: %v", msg.Time.Format("15:04:05.000"), time.Now().Format("15:04:05.000"), msg.From, msg.Text))
				}
			}
			resp.Body.Close()
		}
	}
}
