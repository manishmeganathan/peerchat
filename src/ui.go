package src

import (
	"fmt"
	"io"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const AppVer = "v0.1.0"

type UI struct {
	TerminalApp *tview.Application
	PeerBox     *tview.TextView
	MessageBox  io.Writer

	LogChan   chan string
	InputChan chan string
	SyncChan  chan struct{}
	TermChan  chan struct{}
}

func NewUI() *UI {
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
		SetTitle(fmt.Sprintf("ChatRoom-%s", defaultroom())).
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
		SetLabel(defaultuser() + " > ").
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

				// Check for the user change command
			} else if strings.HasPrefix(line, "/user") {

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
		TerminalApp: app,
		PeerBox:     peerbox,
		MessageBox:  messagebox,
		TermChan:    make(chan struct{}, 1),
	}
}

func (ui *UI) Run() error {
	defer ui.Close()
	return ui.TerminalApp.Run()
}

func (ui *UI) Close() {
	ui.TermChan <- struct{}{}
}

func defaultuser() string {
	return "abbudien"
}

func defaultroom() string {
	return "E52"
}
