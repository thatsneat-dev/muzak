// Command muzak displays the currently playing Apple Music track info
// alongside album artwork rendered via the Kitty graphics protocol
// (supported by Kitty, Ghostty, WezTerm, and other terminals).
// It runs continuously, updating the playback position every second
// and refreshing artwork when a new track begins.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/thatsneat-dev/muzak/internal/music"
	"github.com/thatsneat-dev/muzak/internal/ui"
	"golang.org/x/term"
)

var out = os.Stdout

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	tmp, err := os.CreateTemp("", "tty-graphics-protocol-*.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	artworkPath := tmp.Name()
	tmp.Close()
	defer os.Remove(artworkPath)

	layout := ui.DetectLayout(os.Stdin.Fd(), out.Fd())

	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")

	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)

	keys := make(chan byte, 10)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			keys <- buf[0]
		}
	}()

	a := newApp(ctx, out, artworkPath, layout, keys, winch)
	defer a.cleanup()

	go music.Launch(ctx)

	if on, err := music.ShuffleEnabled(ctx); err == nil {
		a.shuffleOn = on
	}
	if mode, err := music.RepeatMode(ctx); err == nil {
		a.repeatMode = mode
	}

	a.startPoll()
	a.run()
}

// run is the main event loop.
func (a *app) run() {
	for {
		select {
		case <-a.ctx.Done():
			a.quit()
			return
		case result := <-a.pollCh:
			a.pollRunning = false
			if result.err != nil {
				if a.overlayMode == "" {
					fmt.Fprintf(os.Stderr, "muzak: poll: %v\r\n", result.err)
				}
				continue
			}
			a.latestInfo = result.info
			if a.autoBrowsePending && result.info == nil {
				a.autoBrowsePending = false
				a.handlePollInfo(result.info)
				a.openBrowse(true)
			} else {
				if a.autoBrowsePending && result.info != nil {
					a.autoBrowsePending = false
				}
				if a.overlayMode == "" || a.forceRedraw {
					a.handlePollInfo(result.info)
					if a.forceRedraw && a.overlayMode == "queue" {
						a.drawQueue()
					}
					if a.forceRedraw && a.overlayMode == "browse" {
						a.drawBrowse()
					}
					a.forceRedraw = false
				}
			}
		case key := <-a.keys:
			if a.handleKey(key) {
				return
			}
		case <-a.flashTimer.C:
			wasVol := a.flashIcon == "volup" || a.flashIcon == "voldn"
			a.flashIcon = ""
			if wasVol {
				a.drawVolume()
			} else {
				a.redrawControls()
			}
		case <-a.spinnerTicker.C:
			a.spinnerFrame++
			if a.queueLoading && a.overlayMode == "queue" {
				a.drawQueue()
			}
			if a.browseState.IsLoading() && a.overlayMode == "browse" {
				a.drawBrowse()
			}
		case tracks := <-a.queueCh:
			a.queueLoading = false
			a.spinnerTicker.Stop()
			if tracks != nil {
				a.queueTracks = tracks
			}
			if a.overlayMode == "queue" {
				a.drawQueue()
			}
		case items := <-a.playlistsCh:
			a.browseState.PlaylistsLoading = false
			a.browseState.PlaylistsLoaded = true
			if items != nil {
				a.browseState.Playlists = items
			}
			if a.overlayMode == "browse" {
				a.drawBrowse()
			}
		case items := <-a.albumsCh:
			a.browseState.AlbumsLoading = false
			a.browseState.AlbumsLoaded = true
			if items != nil {
				a.browseState.Albums = items
			}
			if a.overlayMode == "browse" {
				a.drawBrowse()
			}
		case result := <-a.albumTracksCh:
			a.browseState.TracksLoading = false
			if result.tracks != nil {
				a.browseState.AlbumTracks = result.tracks
				a.browseState.AlbumTrackCache[result.key] = result.tracks
			}
			if a.overlayMode == "browse" {
				a.drawBrowse()
			}
		case <-a.winch:
			a.layout = ui.DetectLayout(os.Stdin.Fd(), out.Fd())
			a.currentTrack = ""
			a.spaceAllocated = false
			a.hasArtwork = false
			a.forceRedraw = a.overlayMode != ""
			a.allocateSpace()
			a.startPoll()
		case <-a.refresh.C:
			a.startPoll()
		case <-a.ticker.C:
			a.startPoll()
		}
	}
}
