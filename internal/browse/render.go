package browse

import (
	"fmt"
	"strings"

	"github.com/thatsneat-dev/muzak/internal/model"
	"github.com/thatsneat-dev/muzak/internal/ui"
)

// title returns the title for the current screen.
func title(b *State) string {
	switch b.Screen {
	case ScreenPlaylists:
		return ui.IconPlaylistMusic + " Playlists"
	case ScreenFolder:
		if b.FolderName != "" {
			return ui.IconFolder + " " + b.FolderName
		}
		return ui.IconFolder + " Folder"
	case ScreenAlbums:
		return ui.IconLibrary + " Library"
	case ScreenAlbumTracks:
		if b.SelectedAlbum != nil {
			return b.SelectedAlbum.Name
		}
		return "Tracks"
	case ScreenCatalogSearch:
		return ui.IconMusic + " Search Catalog"
	case ScreenCatalogResults:
		return ui.IconMusic + " Results"
	default:
		return "Browse"
	}
}

// subtitle returns an optional subtitle line for the current screen.
func subtitle(b *State) string {
	if b.Screen == ScreenAlbumTracks && b.SelectedAlbum != nil {
		return b.SelectedAlbum.AlbumArtist
	}
	return ""
}

// playlistLabel formats a playlist item, showing a folder icon for folders.
func playlistLabel(p model.Playlist) string {
	if p.IsFolder() {
		return ui.IconFolder + "  " + p.Name
	}
	return fmt.Sprintf("%s \x1b[2m(%d)\x1b[0m", p.Name, p.TrackCount)
}

// itemLabel returns the display label for an item at the given index.
func itemLabel(b *State, idx int) string {
	switch b.Screen {
	case ScreenPlaylists:
		items := VisiblePlaylists(b)
		if idx < len(items) {
			return playlistLabel(items[idx])
		}
	case ScreenFolder:
		items := VisibleFolderItems(b)
		if idx < len(items) {
			return playlistLabel(items[idx])
		}
	case ScreenAlbums:
		items := VisibleAlbums(b)
		if idx < len(items) {
			a := items[idx]
			return fmt.Sprintf("%s — %s", a.Name, a.AlbumArtist)
		}
	case ScreenAlbumTracks:
		items := VisibleAlbumTracks(b)
		if idx < len(items) {
			t := items[idx]
			return fmt.Sprintf("%2d. %s", t.TrackNumber, t.Name)
		}
	case ScreenCatalogResults:
		if idx < len(b.CatalogResults) {
			s := b.CatalogResults[idx]
			return fmt.Sprintf("%s — %s", s.Name, s.Artist)
		}
	default:
		switch idx {
		case RootItemPlaylists:
			return ui.IconPlaylistMusic + "  Playlists"
		case RootItemLibrary:
			return ui.IconLibrary + "  Library"
		case RootItemCatalog:
			return ui.IconMusic + "  Search Catalog"
		}
	}
	return ""
}

// emptyMessage returns the empty-state text for a screen.
func emptyMessage(screen Screen) string {
	switch screen {
	case ScreenPlaylists:
		return "No playlists found."
	case ScreenFolder:
		return "Folder is empty."
	case ScreenAlbums:
		return "No albums in library."
	case ScreenAlbumTracks:
		return "No tracks found."
	case ScreenCatalogResults:
		return "No results found."
	default:
		return ""
	}
}

// Draw renders the browse modal.
func Draw(b *State, layout ui.Layout, spinnerFrame int, hasArtwork bool, drawLine ui.LineDrawer) {
	modalCol := ui.PadLeft
	if hasArtwork {
		modalCol = layout.ArtworkLeft() + layout.ArtworkCols + 2
	}
	modalWidth := 50
	if layout.TerminalCols > modalCol+2 {
		avail := layout.TerminalCols - modalCol - 2
		if avail < modalWidth {
			modalWidth = avail
		}
	}
	if modalWidth < 20 {
		modalWidth = 20
	}
	innerWidth := modalWidth - 4

	visibleRows := layout.ArtworkRows - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Title bar.
	t := title(b)
	sub := subtitle(b)
	titleLen := ui.VisibleLen(t)
	pad := modalWidth - 2 - titleLen - 2
	if pad < 0 {
		pad = 0
	}
	drawLine(ui.PadTop, modalCol,
		"┌ \x1b[1m"+t+"\x1b[0m "+strings.Repeat("─", pad)+"┐")

	list := b.CurrentList()
	count := b.ItemCount()

	// Ensure cursor/scroll are in bounds.
	if list.Cursor >= count && count > 0 {
		list.Cursor = count - 1
	}
	if list.Scroll > list.Cursor {
		list.Scroll = list.Cursor
	}
	if list.Cursor >= list.Scroll+visibleRows {
		list.Scroll = list.Cursor - visibleRows + 1
	}

	// Content rows.
	subtitleRow := -1
	contentStart := 0
	if sub != "" {
		subtitleRow = 0
		contentStart = 1
		visibleRows--
		if visibleRows < 1 {
			visibleRows = 1
		}
	}

	totalRows := visibleRows + contentStart
	for i := range totalRows {
		var content string

		if b.Screen == ScreenCatalogSearch {
			switch i {
			case 0:
				searchLine := "\x1b[33m" + ui.IconSearch + "  " + b.CatalogQuery + "█\x1b[0m"
				content = ui.TruncateVisible(searchLine, innerWidth)
			case 1:
				content = "\x1b[2m" + ui.TruncateVisible("Type a query and press enter", innerWidth) + "\x1b[0m"
			}
		} else if i == subtitleRow {
			content = "\x1b[2m" + ui.TruncateVisible(sub, innerWidth) + "\x1b[0m"
		} else if b.IsLoading() && i == contentStart {
			spinnerChars := [4]string{"⠋", "⠙", "⠹", "⠸"}
			content = spinnerChars[spinnerFrame%len(spinnerChars)] + " Loading…"
		} else if !b.IsLoading() && count == 0 && i == contentStart {
			content = emptyMessage(b.Screen)
		} else {
			idx := list.Scroll + (i - contentStart)
			if idx >= 0 && idx < count {
				label := itemLabel(b, idx)
				cursor := "  "
				if idx == list.Cursor {
					cursor = "\x1b[33m▸\x1b[0m "
				}
				content = cursor + ui.TruncateVisible(label, innerWidth-2)
			}
		}

		visible := ui.VisibleLen(content)
		trailing := innerWidth - visible
		if trailing < 0 {
			trailing = 0
		}
		drawLine(ui.PadTop+1+i, modalCol,
			"│ "+content+strings.Repeat(" ", trailing)+" │")
	}

	// Bottom border with position info.
	bottom := strings.Repeat("─", modalWidth-2)
	if !b.IsLoading() && count > 0 {
		pos := fmt.Sprintf(" %d/%d ", list.Cursor+1, count)
		if len(pos) < modalWidth-2 {
			bottom = strings.Repeat("─", modalWidth-2-len(pos)) + pos
		}
	}
	drawLine(ui.PadTop+1+totalRows, modalCol,
		"└"+bottom+"┘")

	// Search bar.
	if b.SearchActive || b.SearchQuery != "" {
		searchLine := "\x1b[33m" + ui.IconSearch + "  " + b.SearchQuery
		if b.SearchActive {
			searchLine += "█"
		}
		searchLine = ui.TruncateVisible(searchLine, modalWidth)
		searchLine += "\x1b[0m"
		searchPad := modalWidth - ui.VisibleLen(searchLine)
		if searchPad < 0 {
			searchPad = 0
		}
		drawLine(ui.PadTop-1, modalCol, searchLine+strings.Repeat(" ", searchPad))
	} else {
		drawLine(ui.PadTop-1, modalCol, strings.Repeat(" ", modalWidth))
	}

	// Hint lines.
	var hints []string
	if b.Screen == ScreenCatalogSearch {
		hints = []string{"[type] search", "[enter] search", "[esc] back"}
	} else if b.SearchActive {
		hints = []string{"[type] search", "[enter] confirm", "[esc] clear"}
	} else if b.Screen == ScreenRoot {
		hints = []string{"[j|k] move", "[enter] select", "[x] close"}
	} else if b.Screen == ScreenCatalogResults {
		hints = []string{"[j|k] move", "[enter] play", "[/] search", "[h] back", "[x] close"}
	} else {
		hints = []string{"[j|k] move", "[/] search", "[a|d] sort", "[enter] select", "[h] back", "[x] close"}
	}
	ui.DrawHints(hints, modalWidth, ui.PadTop+2+totalRows, modalCol, drawLine)
}
