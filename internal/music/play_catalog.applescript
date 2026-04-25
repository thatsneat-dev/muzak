use framework "Foundation"
use framework "MediaPlayer"
use scripting additions

on run argv
	set trackID to item 1 of argv

	set player to current application's MPMusicPlayerController's systemMusicPlayer()
	set storeIDs to current application's NSArray's arrayWithObject:trackID
	player's setQueueWithStoreIDs:storeIDs
	player's play()
end run
