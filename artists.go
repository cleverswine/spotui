package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zmb3/spotify"
)

func listArtistCategories(n *Node) ([]*Node, error) {
	popularTracksNode := &Node{Label: "Popular Tracks", ID: n.ID, ExpandFunc: listPopularTracks}
	albumsNode := &Node{Label: "Albums", ID: n.ID, ExpandFunc: listAlbums}
	relatedArtistsNode := &Node{Label: "Related Artists", ID: n.ID, ExpandFunc: listRelatedArtists}
	return []*Node{popularTracksNode, albumsNode, relatedArtistsNode}, nil
}

func listRelatedArtists(n *Node) ([]*Node, error) {
	items, err := spoqClient.getRelatedArtists(n.ID)
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		result = append(result, &Node{Name: item.Name, Label: item.Name, ID: item.ID.String(), ExpandFunc: listArtistCategories})
	}
	return result, nil
}

func listAlbums(n *Node) ([]*Node, error) {
	items, err := spoqClient.getAllAlbumsByArtist(n.ID)
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		label := fmt.Sprintf("%s - (%s)", item.Name, item.ReleaseDate)
		if item.AlbumType != "album" {
			label = fmt.Sprintf("%s (%s)", label, item.AlbumType)
		}
		result = append(result, &Node{Name: item.Name, Label: label, ID: item.ID.String(), ExpandFunc: listTracks})
	}
	return result, nil
}

func simpleTrackToNode(item spotify.SimpleTrack, label string) *Node {
	node := &Node{Name: item.Name, Label: label, ID: item.ID.String(), KeyPressFunc: trackKeyPress}
	if libraryContains(item.ID) {
		node.Meta = map[string]interface{}{"color": tcell.ColorLightBlue}
	}
	return node
}

func listPopularTracks(n *Node) ([]*Node, error) {
	items, err := spoqClient.getPopularTracks(n.ID)
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		result = append(result, simpleTrackToNode(item.SimpleTrack, fmt.Sprintf("%s - %s", item.Name, item.Album.Name)))
	}
	return result, nil
}

func listTracks(n *Node) ([]*Node, error) {
	items, err := spoqClient.getAllSongsByAlbum(n.ID)
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		result = append(result, simpleTrackToNode(item, fmt.Sprintf("%2d - %s", item.TrackNumber, item.Name)))
	}
	return result, nil
}

func listArtists(n *Node) ([]*Node, error) {
	items, err := spoqClient.getAllFollowedArtists()
	if err != nil {
		return nil, err
	}
	result := []*Node{}
	for _, item := range items {
		result = append(result, &Node{Name: strings.TrimPrefix(item.Name, "The "), Label: item.Name, ID: item.ID.String(), ExpandFunc: listArtistCategories})
	}
	return result, nil
}

func trackKeyPress(n *Node, k string) {
	playlistChan <- &AddTrackToPlaylist{Track: n, PlaylistIndex: k}
}

func buildArtistTree() *tview.TreeView {
	rootNode := &Node{Label: "Followed Artists", ExpandFunc: listArtists}
	treeRoot := tview.NewTreeNode(rootNode.Label).SetReference(rootNode).SetColor(tcell.ColorGreenYellow).SetSelectable(false)
	tree := tview.NewTreeView().SetRoot(treeRoot).SetCurrentNode(treeRoot)
	tree.SetBorder(true).SetTitle("ARTISTS")
	artists, _ := listArtists(rootNode)
	for _, artist := range artists {
		artist.Level = 1 // sort of a hack to determine if we're at the top level
		treeRoot.AddChild(tview.NewTreeNode(artist.Label).SetReference(artist).SetSelectable(true))
	}
	tree.SetInputCapture(treeKeyBindings(tree))
	return tree
}
