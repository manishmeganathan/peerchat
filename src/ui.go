package src

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Represents the app version
const appversion = "v1.1.0"

// A structure that represents the ChatRoom UI
type UI struct {
	// Represents the ChatRoom (embedded)
	*ChatRoom
	// Represents the tview application
	TerminalApp *tview.Application

	// Represents the user message input queue
	MsgInputs chan string
	// Represents the user command input queue
	CmdInputs chan uicommand

	// Represents the UI element with the list of peers
	peerBox *tview.TextView
	// Represents the UI element with the chat messages and logs
	messageBox *tview.TextView
	// Represents the UI element for the input field
	inputBox *tview.InputField
}

// A structure that represents a UI command
type uicommand struct {
	cmdtype string
	cmdarg  string
}

// A constructor function that generates and
// returns a new UI for a given ChatRoom
func NewUI(cr *ChatRoom) *UI {
	// Create a new Tview App
	app := tview.NewApplication()

	// Initialize the command and message input channels
	cmdchan := make(chan uicommand)
	msgchan := make(chan string)

	// Create a title box
	titlebox := tview.NewTextView().
		SetText(fmt.Sprintf("PeerChat. A P2P Chat Application. %s", appversion)).
		SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter)

	titlebox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen)

	// Create a message box
	messagebox := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	messagebox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle(fmt.Sprintf("ChatRoom-%s", cr.RoomName)).
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite)

	// Create a usage instruction box
	usage := tview.NewTextView().
		SetDynamicColors(true).
		SetText(`[red]/quit[green] - quit the chat | [red]/room <roomname>[green] - change chat room | [red]/user <username>[green] - change user name | [red]/clear[green] - clear the chat`)

	usage.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Usage").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite).
		SetBorderPadding(0, 0, 1, 0)

	// Create peer ID box
	peerbox := tview.NewTextView()

	peerbox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Peers").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite)

	// Create a text input box
	input := tview.NewInputField().
		SetLabel(cr.UserName + " > ").
		SetLabelColor(tcell.ColorGreen).
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack)

	input.SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Input").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite).
		SetBorderPadding(0, 0, 1, 0)

	// Define functionality when the input recieves a done signal (enter/tab)
	input.SetDoneFunc(func(key tcell.Key) {
		// Check if trigger was caused by a Return(Enter) press.
		if key != tcell.KeyEnter {
			return
		}

		// Read the input text
		line := input.GetText()

		// Check if there is any input text. No point printing empty messages
		if len(line) == 0 {
			return
		}

		// Check for command inputs
		if strings.HasPrefix(line, "/") {
			// Split the command
			cmdparts := strings.Split(line, " ")

			// Add a nil arg if there is no argument
			if len(cmdparts) == 1 {
				cmdparts = append(cmdparts, "")
			}

			// Send the command
			cmdchan <- uicommand{cmdtype: cmdparts[0], cmdarg: cmdparts[1]}

		} else {
			// Send the message
			msgchan <- line
		}

		// Reset the input field
		input.SetText("")
	})

	// Create a flexbox to fit all the widgets
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titlebox, 3, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(messagebox, 0, 1, false).
			AddItem(peerbox, 20, 1, false),
			0, 8, false).
		AddItem(input, 3, 1, true).
		AddItem(usage, 3, 1, false)

	// Set the flex as the app root
	app.SetRoot(flex, true)

	// Create UI and return it
	return &UI{
		ChatRoom:    cr,
		TerminalApp: app,
		peerBox:     peerbox,
		messageBox:  messagebox,
		inputBox:    input,
		MsgInputs:   msgchan,
		CmdInputs:   cmdchan,
	}
}

// A method of UI that starts the UI app
func (ui *UI) Run() error {
	go ui.starteventhandler()

	defer ui.Close()
	return ui.TerminalApp.Run()
}

// A method of UI that closes the UI app
func (ui *UI) Close() {
	ui.pscancel()
}

// A method of UI that handles UI events
func (ui *UI) starteventhandler() {
	refreshticker := time.NewTicker(time.Second)
	defer refreshticker.Stop()

	for {
		select {

		case msg := <-ui.MsgInputs:
			// Send the message to outbound queue
			ui.Outbound <- msg
			// Add the message to the message box as a self message
			ui.display_selfmessage(msg)

		case cmd := <-ui.CmdInputs:
			// Handle the recieved command
			go ui.handlecommand(cmd)

		case msg := <-ui.Inbound:
			// Print the recieved messages to the message box
			ui.display_chatmessage(msg)

		case log := <-ui.Logs:
			// Add the log to the message box
			ui.display_logmessage(log)

		case <-refreshticker.C:
			// Refresh the list of peers in the chat room periodically
			ui.syncpeerbox()

		case <-ui.psctx.Done():
			// End the event loop
			return
		}
	}
}

// A method of UI that handles a UI command
func (ui *UI) handlecommand(cmd uicommand) {

	switch cmd.cmdtype {

	// Check for the quit command
	case "/quit":
		// Stop the chat UI
		ui.TerminalApp.Stop()
		return

	// Check for the clear command
	case "/clear":
		// Clear the UI message box
		ui.messageBox.Clear()

	// Check for the room change command
	case "/room":
		if cmd.cmdarg == "" {
			ui.Logs <- chatlog{logprefix: "badcmd", logmsg: "missing room name for command"}
		} else {
			ui.Logs <- chatlog{logprefix: "roomchange", logmsg: fmt.Sprintf("joining new room '%s'", cmd.cmdarg)}

			// Create a reference to the current chatroom
			oldchatroom := ui.ChatRoom

			// Create a new chatroom and join it
			newchatroom, err := JoinChatRoom(ui.Host, ui.UserName, cmd.cmdarg)
			if err != nil {
				ui.Logs <- chatlog{logprefix: "jumperr", logmsg: fmt.Sprintf("could not change chat room - %s", err)}
				return
			}

			// Assign the new chat room to UI
			ui.ChatRoom = newchatroom
			// Sleep for a second to give time for the queues to adapt
			time.Sleep(time.Second * 1)

			// Exit the old chatroom and pause for two seconds
			oldchatroom.Exit()

			// Clear the UI message box
			ui.messageBox.Clear()
			// Update the chat room UI element
			ui.messageBox.SetTitle(fmt.Sprintf("ChatRoom-%s", ui.ChatRoom.RoomName))
		}

	// Check for the user change command
	case "/user":
		if cmd.cmdarg == "" {
			ui.Logs <- chatlog{logprefix: "badcmd", logmsg: "missing user name for command"}
		} else {
			// Update the chat user name
			ui.UpdateUser(cmd.cmdarg)
			// Update the chat room UI element
			ui.inputBox.SetLabel(ui.UserName + " > ")
		}

	// Unsupported command
	default:
		ui.Logs <- chatlog{logprefix: "badcmd", logmsg: fmt.Sprintf("unsupported command - %s", cmd.cmdtype)}
	}
}

// A method of UI that displays a message recieved from a peer
func (ui *UI) display_chatmessage(msg chatmessage) {
	prompt := fmt.Sprintf("[green]<%s>:[-]", msg.SenderName)
	fmt.Fprintf(ui.messageBox, "%s %s\n", prompt, msg.Message)
}

// A method of UI that displays a message recieved from self
func (ui *UI) display_selfmessage(msg string) {
	prompt := fmt.Sprintf("[blue]<%s>:[-]", ui.UserName)
	fmt.Fprintf(ui.messageBox, "%s %s\n", prompt, msg)
}

// A method of UI that displays a log message
func (ui *UI) display_logmessage(log chatlog) {
	prompt := fmt.Sprintf("[yellow]<%s>:[-]", log.logprefix)
	fmt.Fprintf(ui.messageBox, "%s %s\n", prompt, log.logmsg)
}

// A method of UI that refreshes the list of peers
func (ui *UI) syncpeerbox() {
	// Retrieve the list of peers from the chatroom
	peers := ui.PeerList()

	// Clear() is not a threadsafe call
	// So we acquire the thread lock on it
	ui.peerBox.Lock()
	// Clear the box
	ui.peerBox.Clear()
	// Release the lock
	ui.peerBox.Unlock()

	// Iterate over the list of peers
	for _, p := range peers {
		// Generate the pretty version of the peer ID
		peerid := p.Pretty()
		// Shorten the peer ID
		peerid = peerid[len(peerid)-8:]
		// Add the peer ID to the peer box
		fmt.Fprintln(ui.peerBox, peerid)
	}

	// Refresh the UI
	ui.TerminalApp.Draw()
}
