package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const playlistIndexes = "abcdefghijklmnopqrsuvwxyz1234567890"

// AddTrackToPlaylist is an event for adding a track to a playlist
type AddTrackToPlaylist struct {
	Track         *Node
	PlaylistIndex string
}

func listPlaylistTracks(n *Node) ([]*Node, error) {
	items, err := spoqClient.getAllSongsByPlaylist(n.ID)
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		artist := ""
		if len(item.Track.Artists) > 0 {
			artist = item.Track.Artists[0].Name
		}
		label := fmt.Sprintf("%s - %s", artist, item.Track.Name)
		node := &Node{Name: item.Track.Name, Label: label, ID: item.Track.ID.String(), KeyPressFunc: playlistKeyPress}
		node.Meta = map[string]interface{}{"playlistID": n.ID}
		result = append(result, node)
	}
	return result, nil
}

func listPlaylists(n *Node) ([]*Node, error) {
	items, err := spoqClient.getAllPlaylistsForUser()
	if err != nil {
		return nil, err
	}
	// My Library
	libNode := &Node{Name: string(playlistIndexes[0]), Label: "Library", ID: "", ExpandFunc: func(n *Node) ([]*Node, error) {
		result := []*Node{}
		for _, item := range library {
			artist := ""
			if len(item.Artists) > 0 {
				artist = item.Artists[0].Name
			}
			label := fmt.Sprintf("%s - %s", artist, item.Name)
			node := &Node{Name: item.Name, Label: label, ID: item.ID.String(), KeyPressFunc: playlistKeyPress}
			node.Meta = map[string]interface{}{"playlistID": n.ID}
			result = append(result, node)
		}
		return result, nil
	}}
	result := []*Node{libNode}
	// Other user playlists
	for i, item := range items {
		playlistName := string(playlistIndexes[i+1]) // TODO check for out of range...
		result = append(result, &Node{Name: playlistName, Label: playlistName + ") " + item.Name, ID: item.ID.String(), ExpandFunc: listPlaylistTracks})
	}
	return result, nil
}

func playlistKeyPress(n *Node, k string) {
	switch k {
	case "x":
		if n.Meta == nil {
			return
		}
		if playlistID, ok := n.Meta["playlistID"]; ok {
			logger.Printf("removing track \"%s\" from playlist \"%s\"", n.Label, playlistID)
			err := spoqClient.removeTrackFromPlaylist(playlistID.(string), n.ID)
			if err != nil {
				logger.Println(err)
				return
			}
			n.Meta["color"] = tcell.ColorRed
		}
	}
}

func buildPlaylistTree() *tview.TreeView {
	rootNode := &Node{Label: "My Playlists", ExpandFunc: listArtists}
	treeRoot := tview.NewTreeNode(rootNode.Label).SetReference(rootNode).SetColor(tcell.ColorGreenYellow).SetSelectable(false)
	tree := tview.NewTreeView().SetRoot(treeRoot).SetCurrentNode(treeRoot)
	tree.SetBorder(true).SetTitle("PLAYLISTS")
	playlists, _ := listPlaylists(rootNode)
	for _, playlist := range playlists {
		treeRoot.AddChild(tview.NewTreeNode(playlist.Label).SetReference(playlist).SetSelectable(true))
	}
	tree.SetInputCapture(treeKeyBindings(tree))
	// listen for tracks being added
	go func() {
		for e := range playlistChan {
			for _, playlistNode := range treeRoot.GetChildren() {
				playlist := playlistNode.GetReference().(*Node)
				if playlist.Name == e.PlaylistIndex {
					logger.Printf("adding track \"%s\" to playlist \"%s\"", e.Track.Name, playlist.Label)
					err := spoqClient.addTrackToPlaylist(playlist.ID, e.Track.ID)
					if err != nil {
						logger.Println(err)
						return
					}
					newNode := tview.NewTreeNode(e.Track.Name).SetReference(e.Track).
						SetSelectable(true).SetColor(tcell.ColorLightGreen)
					app.QueueUpdateDraw(func() {
						// expand playlist node
						tree.SetCurrentNode(playlistNode)
						f := tree.GetInputCapture()
						f(tcell.NewEventKey(tcell.KeyRight, ' ', tcell.ModNone))
						// add new node at the beginning and select it
						children := playlistNode.GetChildren()
						playlistNode.SetChildren(append([]*tview.TreeNode{newNode}, children...))
						// playlistNode.AddChild(newNode)
						tree.SetCurrentNode(newNode)
					})
					break
				}
			}
		}
	}()
	return tree
}
