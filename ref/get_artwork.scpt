use framework "Foundation"
use framework "MediaPlayer"
use scripting additions

-- Get the system-wide music player
set player to current application's MPMusicPlayerController's systemMusicPlayer()

-- Get item for the current track
set nowPlayingItem to player's nowPlayingItem()

if nowPlayingItem is not missing value then
	-- Get artwork as 500x500px PNG image
	set artworkImage to nowPlayingItem's artwork()'s imageWithSize:{500, 500}
	set imageRep to (artworkImage's representations()'s objectAtIndex:0)
	set artworkData to imageRep's representationUsingType:(current application's NSPNGFileType) |properties|:(missing value)
	
	-- Output to downloads directory
	set filePath to POSIX path of (path to downloads folder) & "artwork.png"
	artworkData's writeToFile:filePath atomically:false
else
	return false
end if
