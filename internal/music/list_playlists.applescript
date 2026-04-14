use framework "Foundation"
use scripting additions

on run argv
	tell application "Music"
		set allPlaylists to every playlist
		set resultArray to current application's NSMutableArray's array()

		repeat with p in allPlaylists
			try
				set cls to class of p
				set isFolder to (cls is folder playlist)
				set isUser to (cls is user playlist)

				if not isFolder and not isUser then
					-- Skip system playlists (library, subscription, etc.)
				else
					set sk to special kind of p

					-- Include folders, regular user playlists, and smart playlists.
					if isFolder or sk is none or sk is smart then
						set skStr to "none"
						if isFolder then
							set skStr to "folder"
						else if sk is smart then
							set skStr to "smart"
						end if

						set dict to current application's NSMutableDictionary's dictionary()
						dict's setValue:(name of p) forKey:"name"
						dict's setValue:(persistent ID of p) forKey:"persistentID"
						dict's setValue:skStr forKey:"specialKind"

						if isFolder then
							dict's setValue:0 forKey:"trackCount"
						else
							dict's setValue:(count of tracks of p) forKey:"trackCount"
						end if

						-- Get parent folder ID if nested inside one.
						try
							set parentPlaylist to parent of p
							if parentPlaylist is not missing value and (class of parentPlaylist is folder playlist) then
								dict's setValue:(persistent ID of parentPlaylist) forKey:"parentID"
							else
								dict's setValue:"" forKey:"parentID"
							end if
						on error
							dict's setValue:"" forKey:"parentID"
						end try

						resultArray's addObject:dict
					end if
				end if
			end try
		end repeat
	end tell

	set jsonData to (current application's NSJSONSerialization's dataWithJSONObject:resultArray options:0 |error|:(missing value))
	set jsonString to (current application's NSString's alloc()'s initWithData:jsonData encoding:(current application's NSUTF8StringEncoding))
	return jsonString as text
end run
