package src

import (
	"context"
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Represents chat room the buffer size for incoming images
const ChatRoomBufffer = 128

// Represents the default room and user names
const defaultuser = "newuser"
const defaultroom = "lobby"

type ChatRoom struct {
	Messages chan *ChatMessage

	ctx          context.Context
	psrouter     *pubsub.PubSub
	pstopic      *pubsub.Topic
	subscription *pubsub.Subscription

	RoomName string
	UserName string
	SelfID   peer.ID
}

type ChatMessage struct {
	Message    string `json:"message"`
	SenderID   string `json:"senderid"`
	SenderName string `json:"sendername"`
}

// A constructor function that generates and returns a new
// ChatRoom for a given P2PHost, username and roomname
func JoinChatRoom(p2phost *P2PHost, username string, roomname string) (*ChatRoom, error) {
	// Alias the PubSub router from the p2phost
	ps := p2phost.PubSubRouter

	// Create a PubSub topic with the room name
	topic, err := ps.Join(roomname)
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

	// Create a ChatRoom object
	chatroom := &ChatRoom{
		ctx:          p2phost.Ctx,
		psrouter:     ps,
		pstopic:      topic,
		subscription: sub,
		RoomName:     roomname,
		UserName:     username,
		SelfID:       p2phost.Host.ID(),
		Messages:     make(chan *ChatMessage),
	}

	// Start the subscription read loop
	go chatroom.ReadLoop()
	// Return the chatroom
	return chatroom, nil
}

// A method of ChatRoom that publishes a ChatMessage
// to the PubSub topic (roomname)
func (cr *ChatRoom) Publish(message string) error {
	// Create a ChatMessage
	m := ChatMessage{
		Message:    message,
		SenderID:   cr.SelfID.Pretty(),
		SenderName: cr.UserName,
	}

	// Marshal the ChatMessage into a JSON
	messagebytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	// Publish the message to the topic and return an error (if any)
	return cr.pstopic.Publish(cr.ctx, messagebytes)
}

// A method of ChatRoom that continously read
// from the subscription until it closes and
// sends it into the message channel
func (cr *ChatRoom) ReadLoop() {
	// Start loop
	for {
		// Read a message from the subscription
		message, err := cr.subscription.Next(cr.ctx)
		// Check error
		if err != nil {
			// Close the messages queue (subscription has closed)
			close(cr.Messages)
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
			continue
		}

		// Send the ChatMessage into the message queue
		cr.Messages <- cm
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
func (cr *ChatRoom) UpdateRoom(roomname string) error {
	// Create a PubSub topic with the room name
	newtopic, err := cr.psrouter.Join(roomname)
	// Check the error
	if err != nil {
		return err
	}

	// Subscribe to the PubSub topic
	newsub, err := newtopic.Subscribe()
	// Check the error
	if err != nil {
		return err
	}

	// Assign the new roomname
	cr.RoomName = roomname
	// Assign the new pubsub topic and subscription
	cr.pstopic = newtopic
	cr.subscription = newsub

	// Return no errors
	return nil
}

// A method of ChatRoom that updates the chat user name
func (cr *ChatRoom) UpdateUser(username string) {
	cr.UserName = username
}
