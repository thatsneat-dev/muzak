use scripting additions

on run argv
	set albumName to item 1 of argv
	set albumArtist to item 2 of argv
	set queueName to "muzak — Album Queue"

	tell application "Music"
		try
			-- Clean up or create the temp playlist
			try
				set queuePlaylist to playlist queueName
				delete every track of queuePlaylist
			on error
				set queuePlaylist to (make new playlist with properties {name:queueName})
			end try

			-- Get album tracks and duplicate to queue
			set albumTracks to every track of library playlist 1 whose album is albumName and album artist is albumArtist
			repeat with t in albumTracks
				duplicate t to queuePlaylist
			end repeat

			play queuePlaylist
		on error errMsg
			error "Could not play album: " & errMsg
		end try
	end tell
end run
