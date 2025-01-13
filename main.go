package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mikerybka/pkg/util"
	"github.com/zmb3/spotify"
)

var (
	spotifyClientID     = util.RequireEnvVar("SPOTIFY_CLIENT_ID")
	spotifyClientSecret = util.RequireEnvVar("SPOTIFY_CLIENT_SECRET")
)

func getSpotifyClient() *spotify.Client {
	redirectURI := "http://localhost:8080/callback"
	auth := spotify.NewAuthenticator(redirectURI, spotify.ScopeUserLibraryRead)
	auth.SetAuthInfo(spotifyClientID, spotifyClientSecret)

	ch := make(chan *spotify.Client)
	state := "spotify_auth_state"

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatalf("Error getting token: %v", err)
			return
		}

		client := auth.NewClient(token)
		fmt.Fprintln(w, "Login Completed! You can close this window.")
		ch <- &client
	})

	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	fmt.Printf("Please log in to Spotify by visiting the following page: %s\n", url)

	client := <-ch
	return client
}

func intPtr(i int) *int {
	return &i
}

func getLikedSongs(client *spotify.Client) []spotify.SavedTrack {
	var allTracks []spotify.SavedTrack
	offset := 0
	limit := 50
	pagenum := 0
	for {
		pagenum++
		fmt.Printf("Page %d...\n", pagenum)
		tracks, err := client.CurrentUsersTracksOpt(&spotify.Options{Offset: intPtr(offset), Limit: intPtr(limit)})
		if err != nil {
			log.Fatalf("failed to fetch liked songs: %v", err)
		}
		allTracks = append(allTracks, tracks.Tracks...)
		if len(tracks.Tracks) < limit {
			break
		}
		offset += limit
	}
	return allTracks
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s", os.Args[0])
	}
	outfile := os.Args[1]

	// Spotify client setup
	spotifyClient := getSpotifyClient()
	fmt.Println("Fetching liked songs from Spotify...")
	likedSongs := getLikedSongs(spotifyClient)
	fmt.Printf("Found %d liked songs.\n", len(likedSongs))

	b, err := json.MarshalIndent(likedSongs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(outfile, b, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("JSON data written to %s.\n", outfile)
}
