package main

import (
	"os"

	"github.com/manishmeganathan/peerchat/src"
	"github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	// username := flag.String("user", "", "username to use in the chatroom.")
	// chatroom := flag.String("room", "", "chatroom to join.")
	// flag.Parse()

	// ctx := context.Background()

	ui := src.NewUI()
	ui.Run()
}
