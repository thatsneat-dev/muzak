use framework "Foundation"
use framework "MediaPlayer"
use scripting additions

on run argv
	set artworkPath to item 1 of argv

	set player to current application's MPMusicPlayerController's systemMusicPlayer()
	set nowPlayingItem to player's nowPlayingItem()

	if nowPlayingItem is missing value then
		return "{\"state\":\"stopped\"}"
	end if

	set trackArtist to nowPlayingItem's artist()
	set trackAlbum to nowPlayingItem's albumTitle()
	set trackName to nowPlayingItem's title()
	set trackDuration to nowPlayingItem's playbackDuration()

	-- Get live playback position and player state from Music.app
	-- (MPMusicPlayerController's currentPlaybackTime does not work correctly on macOS).
	tell application "Music"
		set trackPosition to player position
		if player state is playing then
			set trackState to "playing"
		else
			set trackState to "paused"
		end if
	end tell

	-- Write artwork to provided path (same approach as proven get_artwork.scpt)
	set artworkImage to nowPlayingItem's artwork()'s imageWithSize:{500, 500}
	if artworkImage is not missing value then
		set imageRep to (artworkImage's representations()'s objectAtIndex:0)
		set pngData to imageRep's representationUsingType:(current application's NSPNGFileType) |properties|:(missing value)
		if pngData is not missing value then
			pngData's writeToFile:artworkPath atomically:false
		end if
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
