use scripting additions

on run argv
	set playlistID to item 1 of argv

	tell application "Music"
		try
			set targetPlaylist to first playlist whose persistent ID is playlistID
			play targetPlaylist
		on error errMsg
			error "Could not play playlist: " & errMsg
		end try
	end tell
end run
