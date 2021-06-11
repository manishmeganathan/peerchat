package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func defaultuser() string {
	return "abbudien"
}

func defaultroom() string {
	return "E52"
}

func main() {
	app := tview.NewApplication()

	instructions := tview.NewTextView().
		SetDynamicColors(true).
		SetText(`[red]/quit[green] - quit the chat. [red]/room <roomname>[green] - change chat room. [red]/user <username>[green] - change user name`)

	instructions.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Instructions").
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite).
		SetBorderPadding(0, 0, 1, 0)

	titlebox := tview.NewTextView().
		SetText("PeerChat. A P2P Chat Application. v0.1.0").
		SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter)

	titlebox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen)

	chatbox := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	chatbox.
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle(fmt.Sprintf("ChatRoom-%s", defaultroom())).
		SetTitleAlign(tview.AlignLeft).
		SetTitleColor(tcell.ColorWhite)

	peerbox := tview.NewBox().
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen).
		SetTitle("Peers").
		SetTitleAlign(tview.AlignRight).
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
		if key != tcell.KeyEnter {
			// Not an actual input
			return
		}

		line := input.GetText()

		if len(line) == 0 {
			// Empty Input
			return
		}

		// Check for commands
		if line == "/quit" {
			app.Stop()
			return
		}

		// Reset Input Field
		input.SetText("")
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(titlebox, 3, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(chatbox, 0, 1, false).
			AddItem(peerbox, 20, 1, false),
			0, 1, false).
		AddItem(input, 3, 1, false).
		AddItem(instructions, 3, 1, false)

	if err := app.SetRoot(flex, true).SetFocus(input).Run(); err != nil {
		panic(err)
	}
}
