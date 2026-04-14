use framework "Foundation"
use scripting additions

on run argv
	tell application "Music"
		set resultArray to current application's NSMutableArray's array()

		-- Collect user playlists (regular and smart).
		repeat with p in (every user playlist)
			try
				set sk to special kind of p
				if sk is none or sk is smart then
					if sk is none then
						set skStr to "none"
					else
						set skStr to "smart"
					end if

					set dict to current application's NSMutableDictionary's dictionary()
					dict's setValue:(name of p) forKey:"name"
					dict's setValue:(persistent ID of p) forKey:"persistentID"
					dict's setValue:skStr forKey:"specialKind"
					dict's setValue:(count of tracks of p) forKey:"trackCount"

					try
						set parentPlaylist to parent of p
						if parentPlaylist is not missing value and ((class of parentPlaylist) as text is "folder playlist") then
							dict's setValue:(persistent ID of parentPlaylist) forKey:"parentID"
						else
							dict's setValue:"" forKey:"parentID"
						end if
					on error
						dict's setValue:"" forKey:"parentID"
					end try

					resultArray's addObject:dict
				end if
			end try
		end repeat

		-- Collect folder playlists separately (they are not included in user playlists).
		repeat with p in (every folder playlist)
			try
				set dict to current application's NSMutableDictionary's dictionary()
				dict's setValue:(name of p) forKey:"name"
				dict's setValue:(persistent ID of p) forKey:"persistentID"
				dict's setValue:"folder" forKey:"specialKind"
				dict's setValue:0 forKey:"trackCount"

				try
					set parentPlaylist to parent of p
					if parentPlaylist is not missing value and ((class of parentPlaylist) as text is "folder playlist") then
						dict's setValue:(persistent ID of parentPlaylist) forKey:"parentID"
					else
						dict's setValue:"" forKey:"parentID"
					end if
				on error
					dict's setValue:"" forKey:"parentID"
				end try

				resultArray's addObject:dict
			end try
		end repeat
	end tell

	set jsonData to (current application's NSJSONSerialization's dataWithJSONObject:resultArray options:0 |error|:(missing value))
	set jsonString to (current application's NSString's alloc()'s initWithData:jsonData encoding:(current application's NSUTF8StringEncoding))
	return jsonString as text
end run
