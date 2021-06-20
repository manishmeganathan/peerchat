package src

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Represents chat room the buffer size for incoming images
const ChatRoomBufffer = 128

// Represents the default room and user names
const defaultuser = "newuser"
const defaultroom = "lobby"

type ChatRoom struct {
	Host *P2P

	// Represents the channel of incoming messages
	Messages chan *ChatMessage
	// Represents the channel of logs
	Logs chan uilog

	// Represents the chat room lifecycle context
	psctx    context.Context
	pscancel context.CancelFunc
	// Represents the Pubsub fields
	psrouter     *pubsub.PubSub
	pstopic      *pubsub.Topic
	subscription *pubsub.Subscription

	// Represents the identitiy fields
	RoomName string
	UserName string
	SelfID   peer.ID

	// Represents the message publish queue
	PublishQueue chan string
}

type ChatMessage struct {
	Message    string `json:"message"`
	SenderID   string `json:"senderid"`
	SenderName string `json:"sendername"`
}

// A constructor function that generates and returns a new
// ChatRoom for a given P2PHost, username and roomname
func JoinChatRoom(p2phost *P2P, username string, roomname string) (*ChatRoom, error) {
	// Alias the PubSub router from the p2phost
	ps := p2phost.PubSub

	// Create a PubSub topic with the room name
	topic, err := ps.Join(fmt.Sprintf("room-peerchat-%s", roomname))
	// Check the error
	if err != nil {
		return nil, err
	}

	// Subscribe to the PubSub topic
	sub, err := topic.Subscribe()
	// Check the error
	if err != nil {
		return nil, err
	}

	// Check the provided username
	if username == "" {
		// Use the default user name
		username = defaultuser
	}

	// Check the provided roomname
	if roomname == "" {
		// Use the default room name
		roomname = defaultroom
	}

	pubsubctx, cancel := context.WithCancel(context.Background())

	// Create a ChatRoom object
	chatroom := &ChatRoom{
		Host:         p2phost,
		psctx:        pubsubctx,
		pscancel:     cancel,
		psrouter:     ps,
		pstopic:      topic,
		subscription: sub,
		RoomName:     roomname,
		UserName:     username,
		SelfID:       p2phost.Host.ID(),
		Messages:     make(chan *ChatMessage),
		PublishQueue: make(chan string),
	}

	// Start the subscription read loop
	go chatroom.SubLoop()
	// Start the publish loop
	go chatroom.PubLoop()

	// Return the chatroom
	return chatroom, nil
}

// A method of ChatRoom that publishes a ChatMessage
// to the PubSub topic (roomname)
func (cr *ChatRoom) PubLoop() {
	for {
		select {
		case <-cr.psctx.Done():
			return

		case message := <-cr.PublishQueue:
			// Create a ChatMessage
			m := ChatMessage{
				Message:    message,
				SenderID:   cr.SelfID.Pretty(),
				SenderName: cr.UserName,
			}

			// Marshal the ChatMessage into a JSON
			messagebytes, err := json.Marshal(m)
			if err != nil {
				cr.Logs <- uilog{logprefix: "puberr", logmsg: "could not marshal JSON"}
				continue
			}

			// Publish the message to the topic
			err = cr.pstopic.Publish(cr.psctx, messagebytes)
			if err != nil {
				cr.Logs <- uilog{logprefix: "puberr", logmsg: "could not publish to topic"}
				continue
			}
		}
	}
}

// A method of ChatRoom that continously read
// from the subscription until it closes and
// sends it into the message channel
func (cr *ChatRoom) SubLoop() {
	// Start loop
	for {
		select {
		case <-cr.psctx.Done():
			return

		default:
			// Read a message from the subscription
			message, err := cr.subscription.Next(cr.psctx)
			// Check error
			if err != nil {
				// Close the messages queue (subscription has closed)
				close(cr.Messages)
				cr.Logs <- uilog{logprefix: "suberr", logmsg: "subscription has closed"}
				return
			}

			// Check if message is from self
			if message.ReceivedFrom == cr.SelfID {
				continue
			}

			// Declare a ChatMessage
			cm := &ChatMessage{}
			// Unmarshal the message data into a ChatMessage
			err = json.Unmarshal(message.Data, cm)
			if err != nil {
				cr.Logs <- uilog{logprefix: "suberr", logmsg: "could not unmarshal JSON"}
				continue
			}

			// Send the ChatMessage into the message queue
			cr.Messages <- cm
		}
	}
}

// A method of ChatRoom that returns a list
// of all peer IDs connected to it
func (cr *ChatRoom) PeerList() []peer.ID {
	// Return the slice of peer IDs connected to chat room topic
	return cr.pstopic.ListPeers()
}

// A method of ChatRoom that updates the chat
// room by subscribing to the new topic
func (cr *ChatRoom) Exit() {
	defer cr.pscancel()

	// Cancel the existing subscription
	cr.subscription.Cancel()
	// Close the topic handler
	cr.pstopic.Close()
}

// A method of ChatRoom that updates the chat user name
func (cr *ChatRoom) UpdateUser(username string) {
	cr.UserName = username
}
