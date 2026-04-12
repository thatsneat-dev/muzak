use framework "Foundation"
use framework "MediaPlayer"
use scripting additions

-- Get the system-wide music player
set player to current application's MPMusicPlayerController's systemMusicPlayer()

-- Get item for the current track
set nowPlayingItem to player's nowPlayingItem()

if nowPlayingItem is not missing value then
	set trackArtist to (nowPlayingItem's artist()) as text
	set trackAlbum to (nowPlayingItem's albumTitle()) as text
	set trackName to (nowPlayingItem's title()) as text
	set trackDuration to (nowPlayingItem's playbackDuration()) as real
	tell application "Music" to set trackPosition to player position

	log "Artist: " & trackArtist
	log "Album: " & trackAlbum
	log "Track: " & trackName
	log "Duration: " & formatTime(trackDuration)
	log "Position: " & formatTime(trackPosition)
else
	log "No track is currently playing."
end if

on formatTime(totalSeconds)
	set hrs to (totalSeconds div 3600)
	set mins to ((totalSeconds mod 3600) div 60)
	set secs to (totalSeconds mod 60) as integer
	set hh to text -2 thru -1 of ("0" & hrs)
	set mm to text -2 thru -1 of ("0" & mins)
	set ss to text -2 thru -1 of ("0" & secs)
	return hh & ":" & mm & ":" & ss
end formatTime
