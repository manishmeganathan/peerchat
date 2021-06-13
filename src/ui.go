package src

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const AppVer = "v0.1.0"

type UI struct {
	*ChatRoom
	TerminalApp *tview.Application
	PeerBox     *tview.TextView
	MessageBox  io.Writer

	LogChan   chan string
	InputChan chan string
	SyncChan  chan struct{}
	TermChan  chan struct{}
}

func NewUI(cr *ChatRoom) *UI {
	app := tview.NewApplication()

	syncchan := make(chan struct{})
	inputchan := make(chan string)
	logchan := make(chan string)

	titlebox := tview.NewTextView().
		SetText(fmt.Sprintf("PeerChat. A P2P Chat Application. %s", AppVer)).
		SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter)

	titlebox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen)

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

	usage := tview.NewTextView().
		SetDynamicColors(true).
		SetText(`[red]/quit[green] - quit the chat | [red]/room <roomname>[green] - change chat room | [red]/user <username>[green] - change user name | [red]/sync[green] - refresh`)

	usage.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Usage").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite).
		SetBorderPadding(0, 0, 1, 0)

	peerbox := tview.NewTextView()

	peerbox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Peers").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite)

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

			// Check for quit command
			if strings.HasPrefix(line, "/quit") {
				app.Stop()
				return

				// Check for the sync command
			} else if strings.HasPrefix(line, "/sync") {
				syncchan <- struct{}{}

				// Check for the room change command
			} else if strings.HasPrefix(line, "/room") {
				//TODO: add command handler

				// Check for the user change command
			} else if strings.HasPrefix(line, "/user") {
				//TODO: add command handler

			} else {
				logchan <- "invalid command!"
			}
		}

		// Send the message to the input channel
		inputchan <- line
		// Reset the input field
		input.SetText("")
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titlebox, 3, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(messagebox, 0, 1, false).
			AddItem(peerbox, 20, 1, false),
			0, 8, false).
		AddItem(input, 3, 1, true).
		AddItem(usage, 3, 1, false)

	app.SetRoot(flex, true)

	return &UI{
		ChatRoom:    cr,
		TerminalApp: app,
		PeerBox:     peerbox,
		MessageBox:  messagebox,
		LogChan:     logchan,
		InputChan:   inputchan,
		SyncChan:    syncchan,
		TermChan:    make(chan struct{}, 1),
	}
}

func (ui *UI) Run() error {
	go ui.starteventhandler()

	defer ui.Close()
	return ui.TerminalApp.Run()
}

func (ui *UI) Close() {
	ui.TermChan <- struct{}{}
}

func (ui *UI) starteventhandler() {
	refreshticker := time.NewTicker(time.Second)
	defer refreshticker.Stop()

	for {
		select {

		case input := <-ui.InputChan:
			err := ui.ChatRoom.Publish(input)

			if err != nil {
				ui.display_logmessage("error", "message publish failed!")
			}

			ui.display_selfmessage(input)

		case msg := <-ui.ChatRoom.Messages:
			// when we receive a message from the chat room, print it to the message window
			ui.display_chatmessage(msg)

		case <-refreshticker.C:
			// refresh the list of peers in the chat room periodically
			ui.syncpeerbox()

		case <-ui.ChatRoom.ctx.Done():
			return

		case <-ui.TermChan:
			return

			//TODO: add command handler
			//TODO: add log chan handler
			//TODO: add sync chan handler
		}
	}
}

func (ui *UI) display_chatmessage(msg *ChatMessage) {
	prompt := fmt.Sprintf("[green]<%s>:[-]", msg.SenderName)
	fmt.Fprintf(ui.MessageBox, "%s %s\n", prompt, msg.Message)
}

func (ui *UI) display_selfmessage(msg string) {
	prompt := fmt.Sprintf("[blue]<%s>:[-]", ui.ChatRoom.UserName)
	fmt.Fprintf(ui.MessageBox, "%s %s\n", prompt, msg)
}

func (ui *UI) display_logmessage(prefix, log string) {
	prompt := fmt.Sprintf("[yellow]<%s>:[-]", prefix)
	fmt.Fprintf(ui.MessageBox, "%s %s\n", prompt, log)
}

func (ui *UI) syncpeerbox() {
	peers := ui.ChatRoom.PeerList()
	ui.display_logmessage("log", fmt.Sprintf("found peers - %v", peers))

	// clear is not threadsafe so we need to take the lock.
	ui.PeerBox.Lock()
	ui.PeerBox.Clear()
	ui.PeerBox.Unlock()

	for _, p := range peers {
		peerid := p.Pretty()
		peerid = peerid[len(peerid)-8:]
		fmt.Fprintln(ui.PeerBox, peerid)
	}

	ui.TerminalApp.Draw()
}
