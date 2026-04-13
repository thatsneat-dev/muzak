use framework "Foundation"
use scripting additions

on run argv
	tell application "Music"
		set allAlbums to album of every track of library playlist 1
		set allArtists to album artist of every track of library playlist 1
	end tell

	set trackCount to count of allAlbums
	set albumMap to current application's NSMutableDictionary's dictionary()

	repeat with i from 1 to trackCount
		set aName to item i of allAlbums
		set aArtist to item i of allArtists
		if aName is missing value then set aName to ""
		if aArtist is missing value then set aArtist to ""

		set mapKey to (aArtist & "|||" & aName)

		set existing to (albumMap's objectForKey:mapKey)
		if existing is missing value then
			set dict to current application's NSMutableDictionary's dictionary()
			dict's setValue:aName forKey:"name"
			dict's setValue:aArtist forKey:"albumArtist"
			dict's setValue:1 forKey:"trackCount"
			albumMap's setObject:dict forKey:mapKey
		else
			set curCount to (existing's objectForKey:"trackCount")'s integerValue()
			existing's setValue:(curCount + 1) forKey:"trackCount"
		end if
	end repeat

	set resultArray to albumMap's allValues()'s sortedArrayUsingDescriptors:{current application's NSSortDescriptor's sortDescriptorWithKey:"albumArtist" ascending:true selector:"localizedCaseInsensitiveCompare:", current application's NSSortDescriptor's sortDescriptorWithKey:"name" ascending:true selector:"localizedCaseInsensitiveCompare:"}

	set jsonData to (current application's NSJSONSerialization's dataWithJSONObject:resultArray options:0 |error|:(missing value))
	set jsonString to (current application's NSString's alloc()'s initWithData:jsonData encoding:(current application's NSUTF8StringEncoding))
	return jsonString as text
end run
