package main

import (
	"sort"
	"strings"

	"github.com/zmb3/spotify"
)

// byArtistName assists in sorting artists by name
type byArtistName []spotify.FullArtist

func (a byArtistName) Len() int      { return len(a) }
func (a byArtistName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byArtistName) Less(i, j int) bool {
	return strings.TrimPrefix(a[i].Name, "The ") < strings.TrimPrefix(a[j].Name, "The ")
}

// byAlbumYear assists in sorting albums by year
type byAlbumYear []spotify.SimpleAlbum

func (a byAlbumYear) Len() int      { return len(a) }
func (a byAlbumYear) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byAlbumYear) Less(i, j int) bool {
	return a[i].ReleaseDateTime().Before(a[j].ReleaseDateTime())
}

// byPlaylistTrack assists in sorting tracks by artist / name
type byPlaylistTrack []spotify.PlaylistTrack

func (a byPlaylistTrack) Len() int      { return len(a) }
func (a byPlaylistTrack) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPlaylistTrack) Less(i, j int) bool {
	return strings.TrimPrefix(a[i].Track.Artists[0].Name, "The ")+a[i].Track.Name < strings.TrimPrefix(a[j].Track.Artists[0].Name, "The ")+a[j].Track.Name
}

// byPlaylistTrack assists in sorting tracks by artist / name
type bySavedTrack []spotify.SavedTrack

func (a bySavedTrack) Len() int      { return len(a) }
func (a bySavedTrack) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a bySavedTrack) Less(i, j int) bool {
	return strings.TrimPrefix(a[i].Artists[0].Name, "The ")+a[i].Name < strings.TrimPrefix(a[j].Artists[0].Name, "The ")+a[j].Name
}

// Client wraps the github.com/zmb3/spotify with higher level utility funcs
type Client struct {
	spotifyClient *spotify.Client
}

// NewSpoqClient creates a SpoqClient using the provided spotify client
func NewSpoqClient(client *spotify.Client) *Client {
	return &Client{spotifyClient: client}
}

func (c *Client) removeTrackFromPlaylist(id string, track string) error {
	if id == "" {
		err := c.spotifyClient.RemoveTracksFromLibrary(spotify.ID(track))
		return err
	}
	_, err := c.spotifyClient.RemoveTracksFromPlaylist(spotify.ID(id), spotify.ID(track))
	return err
}

func (c *Client) addTrackToPlaylist(id string, track string) error {
	if id == "" {
		err := c.spotifyClient.AddTracksToLibrary(spotify.ID(track))
		return err
	}
	_, err := c.spotifyClient.AddTracksToPlaylist(spotify.ID(id), spotify.ID(track))
	return err
}

func (c *Client) getAllSavedTracks() ([]spotify.SavedTrack, error) {
	all := []spotify.SavedTrack{}
	page := 1
	limit := 50
	for {
		offset := (page - 1) * limit
		items, err := c.spotifyClient.CurrentUsersTracksOpt(&spotify.Options{Limit: &limit, Offset: &offset})
		if err != nil {
			return nil, err
		}
		all = append(all, items.Tracks...)
		if items.Next == "" {
			break
		}
		page = page + 1
	}
	sort.Sort(bySavedTrack(all))
	return all, nil
}

func (c *Client) getAllPlaylistsForUser() ([]spotify.SimplePlaylist, error) {
	user, err := c.spotifyClient.CurrentUser()
	if err != nil {
		return nil, err
	}
	all := []spotify.SimplePlaylist{}
	page := 1
	limit := 50
	for {
		offset := (page - 1) * limit
		items, err := c.spotifyClient.CurrentUsersPlaylistsOpt(&spotify.Options{Limit: &limit, Offset: &offset})
		if err != nil {
			return nil, err
		}
		for _, item := range items.Playlists {
			if item.Owner.ID == user.ID {
				all = append(all, item)
			}
		}
		if items.Next == "" {
			break
		}
		page = page + 1
	}
	return all, nil
}

func (c *Client) getAllSongsByPlaylist(id string) ([]spotify.PlaylistTrack, error) {
	all := []spotify.PlaylistTrack{}
	page := 1
	limit := 50
	for {
		offset := (page - 1) * limit
		items, err := c.spotifyClient.GetPlaylistTracksOpt(spotify.ID(id), &spotify.Options{Limit: &limit, Offset: &offset}, "")
		if err != nil {
			return nil, err
		}
		all = append(all, items.Tracks...)
		if items.Next == "" {
			break
		}
		page = page + 1
	}
	sort.Sort(byPlaylistTrack(all))
	return all, nil
}

func (c *Client) getAllSongsByAlbum(id string) ([]spotify.SimpleTrack, error) {
	all := []spotify.SimpleTrack{}
	page := 1
	limit := 50
	for {
		offset := (page - 1) * limit
		items, err := c.spotifyClient.GetAlbumTracksOpt(spotify.ID(id), &spotify.Options{Limit: &limit, Offset: &offset})
		if err != nil {
			return nil, err
		}
		all = append(all, items.Tracks...)
		if items.Next == "" {
			break
		}
		page = page + 1
	}
	return all, nil
}

func (c *Client) getAllAlbumsByArtist(id string) ([]spotify.SimpleAlbum, error) {
	all := []spotify.SimpleAlbum{}
	albumTypes := spotify.AlbumTypeAlbum | spotify.AlbumTypeSingle
	page := 1
	limit := 50
	//country := spotify.CountryUSA
	for {
		offset := (page - 1) * limit
		items, err := c.spotifyClient.GetArtistAlbumsOpt(spotify.ID(id), &spotify.Options{Limit: &limit, Offset: &offset}, albumTypes)
		if err != nil {
			return nil, err
		}
		all = append(all, items.Albums...)
		if items.Next == "" {
			break
		}
		page = page + 1
	}
	sort.Sort(byAlbumYear(all))
	return all, nil
}

func (c *Client) getRelatedArtists(id string) ([]spotify.FullArtist, error) {
	return c.spotifyClient.GetRelatedArtists(spotify.ID(id))
}

func (c *Client) getAllFollowedArtists() ([]spotify.FullArtist, error) {
	all := []spotify.FullArtist{}
	next := ""
	for {
		items, err := c.spotifyClient.CurrentUsersFollowedArtistsOpt(-1, next)
		if err != nil {
			return nil, err
		}
		all = append(all, items.Artists...)
		next = items.Cursor.After
		if next == "" {
			break
		}
	}
	sort.Sort(byArtistName(all))
	return all, nil
}

func (c *Client) getPopularTracks(id string) ([]spotify.FullTrack, error) {
	return c.spotifyClient.GetArtistsTopTracks(spotify.ID(id), spotify.CountryUSA)
}
