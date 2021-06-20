package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/manishmeganathan/peerchat/src"
	"github.com/sirupsen/logrus"
)

const figlet = `

W E L C O M E  T O
					     db                  db   
					     88                  88   
.8d888b. .d8888b. .d8888b. .d8888b. .d8888b. 88d888b. .d8888b. d8888P 
88'  '88 88ooood8 88ooood8 88'  '88 88'      88'  '88 88'  '88   88   
88.  .88 88.      88.      88       88.      88    88 88.  .88   88   
888888P' '88888P' '88888P' db       '88888P' db    db '8888888   '88P   
88                                                                    
dP     
`

func init() {
	// Log as Text with color
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC822,
	})

	// Log to stdout
	logrus.SetOutput(os.Stdout)
}

func main() {
	// Define input flags
	username := flag.String("user", "", "username to use in the chatroom.")
	chatroom := flag.String("room", "", "chatroom to join.")
	loglevel := flag.String("log", "", "level of logs to print.")
	discovery := flag.String("discover", "", "method to use for discovery.")
	// Parse input flags
	flag.Parse()

	// Set the log level
	switch *loglevel {
	case "panic", "PANIC":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal", "FATAL":
		logrus.SetLevel(logrus.FatalLevel)
	case "error", "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn", "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "info", "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug", "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace", "TRACE":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Display the welcome figlet
	fmt.Println(figlet)
	fmt.Println("The PeerChat Application is starting.")
	fmt.Println("This may take upto 30 seconds.")
	fmt.Println()

	// Create a new P2PHost
	p2phost := src.NewP2P()
	logrus.Infoln("Completed P2P Setup")

	// Connect to peers with the chosen discovery method
	switch *discovery {
	case "announce":
		p2phost.AnnounceConnect()
	case "advertise":
		p2phost.AdvertiseConnect()
	default:
		p2phost.AdvertiseConnect()
	}
	logrus.Infoln("Connected to Service Peers")

	// Join the chat room
	chatapp, _ := src.JoinChatRoom(p2phost, *username, *chatroom)
	logrus.Infof("Joined the '%s' chatroom as '%s'", *chatroom, *username)

	// Wait for network setup to complete
	time.Sleep(time.Second * 5)

	// Create the Chat UI
	ui := src.NewUI(chatapp)
	// Start the UI system
	ui.Run()
}
