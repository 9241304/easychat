package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type messageWithRecipients struct {
	Message    Message
	Recipients map[string]struct{}
}

type userNotifiers struct {
	GetUpdatesCh      chan struct{}
	GetUpdatesCounter int
}

var users = make(map[string]*userNotifiers)
var messages []*messageWithRecipients
var mutex sync.Mutex

func LaunchServer() {
	http.HandleFunc("/postMessage", postMessage)
	http.HandleFunc("/getUpdates", getUpdates)

	http.ListenAndServe(":14300", nil)
}

//should be called after mutex lock
func getRecipientsFromUsers() map[string]struct{} {
	res := make(map[string]struct{})
	for user, _ := range users {
		res[user] = struct{}{}
	}

	return res
}

func createMessageForAllUsersAndNotify(from, text string) {
	recipients := getRecipientsFromUsers()
	messages = append(messages, &messageWithRecipients{
		Message: Message{
			Text: text,
			From: from,
			Time: time.Now(),
		},
		Recipients: recipients,
	})

	for user, notifiers := range users {
		if notifiers.GetUpdatesCh != nil {
			close(notifiers.GetUpdatesCh)
			users[user].GetUpdatesCh = nil
		}
	}
}

func leave(w http.ResponseWriter, req *http.Request) {

}

func postMessage(w http.ResponseWriter, req *http.Request) {
	var pmr PostMessageRequest
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&pmr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		if len(users) == 0 {
			http.Error(w, "No users in chat", http.StatusBadRequest)
		} else {
			mutex.Lock()
			createMessageForAllUsersAndNotify(pmr.From, pmr.Text)
			mutex.Unlock()
		}
	}
}

//TODO check if request from one user sent twice
func getUpdates(w http.ResponseWriter, req *http.Request) {
	forUser := req.URL.Query().Get("for")
	if len(forUser) == 0 {
		http.Error(w, "Invalid user", http.StatusBadRequest)
	} else {
		mutex.Lock()
		if _, ok := users[forUser]; !ok {
			users[forUser] = &userNotifiers{}
			createMessageForAllUsersAndNotify("SERVER", fmt.Sprintf("User %v entered to chat", forUser))
		}
		users[forUser].GetUpdatesCounter++
		mutex.Unlock()

		res := getMessagesFor(forUser)
		if len(res) == 0 {
			mutex.Lock()
			ch := make(chan struct{})
			users[forUser].GetUpdatesCh = ch
			mutex.Unlock()

			select {
			case <-req.Context().Done():
			case <-ch:
			case <-time.After(60 * time.Second):
			}
			res = getMessagesFor(forUser)
		}

		json.NewEncoder(w).Encode(&GetUpdatesReply{res})

		go func() {
			time.Sleep(5 * time.Second)
			mutex.Lock()
			users[forUser].GetUpdatesCounter--
			if users[forUser].GetUpdatesCounter == 0 {
				delete(users, forUser)
				createMessageForAllUsersAndNotify("SERVER", fmt.Sprintf("User %v exited from chat", forUser))
			}
			mutex.Unlock()
		}()
	}
}

func getMessagesFor(user string) []*Message {
	var res []*Message
	mutex.Lock()
	defer mutex.Unlock()
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		if _, ok := msg.Recipients[user]; !ok {
			continue
		}

		res = append(res, &msg.Message)
		if len(msg.Recipients) == 1 {
			messages = append(messages[:i], messages[i+1:]...)
			i--
		} else {
			delete(msg.Recipients, user)
		}
	}

	return res
}
