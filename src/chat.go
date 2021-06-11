package src

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Represents chat room the buffer size for incoming images
const ChatRoomBufffer = 128

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
	//go chatroom.ReadLoop()
	// Return the chatroom
	return chatroom, nil
}
