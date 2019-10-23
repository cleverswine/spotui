package main

import (
	"log"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/zmb3/spotify"
)

var spoqClient *Client
var logger *log.Logger
var app *tview.Application
var library []spotify.SavedTrack
var playlistChan chan *AddTrackToPlaylist

func main() {
	// get an authenticated Spotify client
	spotifyClientBuilder := NewSpotifyClientBuilder(nil)
	spotifyClient, err := spotifyClientBuilder.GetClient()
	if err != nil {
		log.Fatal(err)
	}
	defer spotifyClientBuilder.SaveToken(spotifyClient)

	// client wrapper for high level utils, paging, etc
	spoqClient = NewSpoqClient(spotifyClient)
	library, err = spoqClient.getAllSavedTracks()
	if err != nil {
		log.Fatal(err)
	}

	// TUI app
	app = tview.NewApplication()

	// bottom pane for logging
	bottom := tview.NewTextView().SetDynamicColors(true).SetRegions(true).SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	bottom.SetTitle("LOG").SetBorder(true)
	logger = log.New(bottom, "", log.Ltime)

	// trees
	playlistChan = make(chan *AddTrackToPlaylist)
	defer close(playlistChan)
	playlistTree := buildPlaylistTree()
	artistTree := buildArtistTree()

	// app level key bindings
	app.SetInputCapture(appKeyBindings(app, artistTree, playlistTree))

	// layout
	flex := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(artistTree, 0, 1, true).
				AddItem(playlistTree, 0, 1, false), 0, 3, true).
			AddItem(bottom, 0, 1, true), 0, 1, false)

	// run
	if err := app.SetRoot(flex, true).SetFocus(artistTree).Run(); err != nil {
		panic(err)
	}
}

func libraryContains(id spotify.ID) bool {
	for i := 0; i < len(library); i++ {
		if library[i].ID == id {
			return true
		}
	}
	return false
}

func appKeyBindings(app *tview.Application, artistTree *tview.TreeView, playlistTree *tview.TreeView) func(key *tcell.EventKey) *tcell.EventKey {
	return func(key *tcell.EventKey) *tcell.EventKey {
		if key.Rune() == 'q' {
			app.Stop()
			return nil
		}
		switch key.Key() {
		case tcell.KeyTab:
			if artistTree.HasFocus() {
				app.SetFocus(playlistTree)
			} else {
				app.SetFocus(artistTree)
			}
			return nil
		}
		return key
	}
}
