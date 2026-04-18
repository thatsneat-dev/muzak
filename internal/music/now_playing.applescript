use framework "Foundation"
use framework "MediaPlayer"
use scripting additions

on run argv
	set artworkPath to item 1 of argv

	set player to current application's MPMusicPlayerController's systemMusicPlayer()
	set nowPlayingItem to player's nowPlayingItem()

	-- Source 1: MPMusicPlayerController (works for library tracks)
	if nowPlayingItem is not missing value then
		set trackArtist to nowPlayingItem's artist()
		set trackAlbum to nowPlayingItem's albumTitle()
		set trackName to nowPlayingItem's title()
		set trackDuration to nowPlayingItem's playbackDuration()

		if trackArtist is missing value then set trackArtist to ""
		if trackAlbum is missing value then set trackAlbum to ""
		if trackName is missing value then set trackName to ""
		if trackDuration is missing value then set trackDuration to 0

		tell application "Music"
			set trackPosition to player position
			if player state is playing then
				set trackState to "playing"
			else
				set trackState to "paused"
			end if
		end tell

		-- Write artwork via MediaPlayer framework
		set artworkImage to nowPlayingItem's artwork()'s imageWithSize:{500, 500}
		if artworkImage is not missing value then
			set imageRep to (artworkImage's representations()'s objectAtIndex:0)
			set pngData to imageRep's representationUsingType:(current application's NSPNGFileType) |properties|:(missing value)
			if pngData is not missing value then
				pngData's writeToFile:artworkPath atomically:false
			end if
		end if

	else
		-- Source 2: Music.app scripting (fallback for streaming/catalog tracks)
		tell application "Music"
			if player state is stopped then
				return "{\"state\":\"stopped\"}"
			end if

			set trackName to name of current track
			set trackArtist to artist of current track
			set trackAlbum to album of current track
			set trackDuration to duration of current track
			set trackPosition to player position

			if player state is playing then
				set trackState to "playing"
			else
				set trackState to "paused"
			end if

			-- Write artwork via Music.app scripting
			try
				set artworkData to raw data of artwork 1 of current track
				set fileRef to open for access POSIX file artworkPath with write permission
				set eof of fileRef to 0
				write artworkData to fileRef
				close access fileRef
			on error
				try
					close access POSIX file artworkPath
				end try
			end try
		end tell
	end if

	-- Build JSON response
	set dict to current application's NSMutableDictionary's dictionary()
	dict's setValue:trackArtist forKey:"artist"
	dict's setValue:trackAlbum forKey:"album"
	dict's setValue:trackName forKey:"name"
	dict's setValue:trackDuration forKey:"duration"
	dict's setValue:trackPosition forKey:"position"
	dict's setValue:trackState forKey:"state"

	set jsonData to (current application's NSJSONSerialization's dataWithJSONObject:dict options:0 |error|:(missing value))
	set jsonString to (current application's NSString's alloc()'s initWithData:jsonData encoding:(current application's NSUTF8StringEncoding))
	return jsonString as text
end run
