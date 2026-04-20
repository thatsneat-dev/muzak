// Package ui provides terminal rendering primitives, layout computation,
// ANSI text helpers, Nerd Font icon constants, Kitty graphics protocol support,
// and drawing functions for the muzak TUI.
package ui

import (
	"os"

	"golang.org/x/term"
)

const (
	DefaultArtworkCols = 20
	DefaultArtworkRows = 10
	DefaultBarWidth    = 30

	PadTop  = 1
	PadLeft = 2

	VolBarWidth = 1
	VolBarGap   = 2

	NumTextLines = 5

	RestartThreshold = 3.0
	MaxQueueTracks   = 10
)

// Layout holds computed terminal layout dimensions.
type Layout struct {
	TerminalCols int
	ArtworkCols  int
	ArtworkRows  int
}

// DisplayRows returns the total rows used by the display area.
func (l Layout) DisplayRows() int {
	return PadTop + l.ArtworkRows
}

// ArtworkLeft returns the column where artwork starts.
func (l Layout) ArtworkLeft() int {
	return PadLeft + VolBarWidth + VolBarGap
}

// TextCol returns the column where text starts.
func (l Layout) TextCol(withArtwork bool) int {
	if withArtwork {
		return l.ArtworkLeft() + l.ArtworkCols + 2
	}
	return PadLeft
}

// TextWidth returns the available width for text.
func (l Layout) TextWidth(withArtwork bool) int {
	col := l.TextCol(withArtwork)
	if l.TerminalCols <= col {
		return 0
	}
	return l.TerminalCols - col
}

// BarWidth returns the progress bar width, constrained by available space.
func (l Layout) BarWidth(withArtwork bool) int {
	bw := DefaultBarWidth
	if l.TerminalCols > 0 {
		avail := l.TerminalCols - l.TextCol(withArtwork) - 2
		if avail < bw && avail > 0 {
			bw = avail
		}
	}
	return bw
}

// ModalCol returns the column where a modal starts.
func (l Layout) ModalCol(withArtwork bool) int {
	return l.TextCol(withArtwork)
}

// DetectLayout reads the terminal size and computes layout dimensions.
func DetectLayout(stdinFd, outFd uintptr) Layout {
	w, h, err := term.GetSize(int(stdinFd))
	if err != nil {
		w, h, err = term.GetSize(int(outFd))
	}
	return ComputeLayout(w, h, err)
}

// ComputeLayout computes layout dimensions from terminal size.
func ComputeLayout(w, h int, err error) Layout {
	l := Layout{
		ArtworkCols: DefaultArtworkCols,
		ArtworkRows: DefaultArtworkRows,
	}
	if err != nil || h <= 0 {
		return l
	}
	l.TerminalCols = w
	if h < PadTop+DefaultArtworkRows+2 {
		l.ArtworkRows = h - 2
		if l.ArtworkRows < 1 {
			l.ArtworkRows = 1
		}
	}
	l.ArtworkCols = l.ArtworkRows * 2
	return l
}

// TerminalSize reads the terminal dimensions.
func TerminalSize() (cols, rows int, err error) {
	cols, rows, err = term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		cols, rows, err = term.GetSize(int(os.Stdout.Fd()))
	}
	return
}
