package src

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

const serviceCID = "manishmeganathan/peerchat"

// A structure that represents P2P Host
type P2PHost struct {
	// Represents the host context layer
	Ctx context.Context

	// Represents the libp2p host
	Host host.Host

	// Represents the DHT routing table
	KadDHT *dht.IpfsDHT

	// Represents the PubSub router
	PubSubRouter *pubsub.PubSub
}

/*
A constructor function that generates and returns a P2PHost for a given context object.

Constructs a libp2p host with a multiaddr on 0.0.0.0/0 IPV4 address and configure it
with NATPortMap to open a port in the firewall using UPnP. A GossipSub pubsub router
is initialized for transport and a Kademlia DHT for peer discovery
*/
func NewP2PHost(ctx context.Context) *P2PHost {
	// Create a new multiaddr object
	sourcemultiaddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	// Construct a new LibP2P host with the multiaddr and the NAT Port Map
	libhost, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourcemultiaddr),
		libp2p.NATPortMap(),
	)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Host Creation Failed!")
	}

	// Create a new PubSub service which uses a GossipSub router
	gossip, err := pubsub.NewGossipSub(ctx, libhost)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("GossipSub Router Creation Failed!")
	}

	// Bind the LibP2P host to a Kademlia DHT peer
	kaddht, err := dht.New(ctx, libhost, dht.Mode(dht.ModeServer))
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Kademlia DHT Creation Failed!")
	}

	// Return the P2PHost
	return &P2PHost{
		Ctx:          ctx,
		Host:         libhost,
		KadDHT:       kaddht,
		PubSubRouter: gossip,
	}
}
