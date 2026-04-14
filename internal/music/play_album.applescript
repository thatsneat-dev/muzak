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

			-- Get album tracks sorted by disc and track number
			set albumTracks to (every track of library playlist 1 whose album is albumName and album artist is albumArtist)
			set trackCount to count of albumTracks

			-- Bubble sort by disc number then track number
			repeat with i from 1 to trackCount - 1
				repeat with j from 1 to trackCount - i
					set dA to disc number of item j of albumTracks
					set dB to disc number of item (j + 1) of albumTracks
					set tA to track number of item j of albumTracks
					set tB to track number of item (j + 1) of albumTracks
					if (dA > dB) or (dA = dB and tA > tB) then
						set tmp to item j of albumTracks
						set item j of albumTracks to item (j + 1) of albumTracks
						set item (j + 1) of albumTracks to tmp
					end if
				end repeat
			end repeat

			repeat with t in albumTracks
				duplicate t to queuePlaylist
			end repeat

			play queuePlaylist
		on error errMsg
			error "Could not play album: " & errMsg
		end try
	end tell
end run
