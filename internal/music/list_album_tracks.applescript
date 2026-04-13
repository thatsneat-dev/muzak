use framework "Foundation"
use scripting additions

on run argv
	set albumName to item 1 of argv
	set albumArtist to item 2 of argv

	tell application "Music"
		set albumTracks to every track of library playlist 1 whose album is albumName and album artist is albumArtist
		set resultArray to current application's NSMutableArray's array()

		repeat with t in albumTracks
			try
				set dict to current application's NSMutableDictionary's dictionary()
				dict's setValue:(name of t) forKey:"name"
				dict's setValue:(persistent ID of t) forKey:"persistentID"
				dict's setValue:(disc number of t) forKey:"discNumber"
				dict's setValue:(track number of t) forKey:"trackNumber"
				dict's setValue:(duration of t) forKey:"duration"
				resultArray's addObject:dict
			end try
		end repeat
	end tell

	set sortDescriptors to {current application's NSSortDescriptor's sortDescriptorWithKey:"discNumber" ascending:true, current application's NSSortDescriptor's sortDescriptorWithKey:"trackNumber" ascending:true}
	set sortedArray to resultArray's sortedArrayUsingDescriptors:sortDescriptors

	set jsonData to (current application's NSJSONSerialization's dataWithJSONObject:sortedArray options:0 |error|:(missing value))
	set jsonString to (current application's NSString's alloc()'s initWithData:jsonData encoding:(current application's NSUTF8StringEncoding))
	return jsonString as text
end run
