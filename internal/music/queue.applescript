use scripting additions

on run argv
	set maxUpcoming to 10

	tell application "Music"
		if player state is stopped then
			return "{\"tracks\":[]}"
		end if

		set cp to current playlist
		set ct to current track
		set ctID to id of ct
		set totalTracks to count of tracks of cp

		set foundPos to 0
		repeat with i from 1 to totalTracks
			if id of track i of cp is ctID then
				set foundPos to i
				exit repeat
			end if
		end repeat

		if foundPos is 0 or foundPos is totalTracks then
			return "{\"tracks\":[]}"
		end if

		set lastIndex to foundPos + maxUpcoming
		if lastIndex > totalTracks then set lastIndex to totalTracks

		set jsonParts to {}
		repeat with i from (foundPos + 1) to lastIndex
			set t to track i of cp
			set tName to name of t
			set tArtist to artist of t
			set tAlbum to album of t
			set tDuration to duration of t

			if tName is missing value then set tName to ""
			if tArtist is missing value then set tArtist to ""
			if tAlbum is missing value then set tAlbum to ""
			if tDuration is missing value then set tDuration to 0

			-- Escape quotes in strings
			set tName to my escapeJSON(tName)
			set tArtist to my escapeJSON(tArtist)
			set tAlbum to my escapeJSON(tAlbum)

			set end of jsonParts to "{\"name\":\"" & tName & "\",\"artist\":\"" & tArtist & "\",\"album\":\"" & tAlbum & "\",\"duration\":" & tDuration & "}"
		end repeat

		set AppleScript's text item delimiters to ","
		set jsonArray to jsonParts as text
		set AppleScript's text item delimiters to ""

		return "{\"tracks\":[" & jsonArray & "]}"
	end tell
end run

on toHexByte(n)
	set hexChars to "0123456789ABCDEF"
	set highDigit to (n div 16) + 1
	set lowDigit to (n mod 16) + 1
	return character highDigit of hexChars & character lowDigit of hexChars
end toHexByte

on escapeJSON(str)
	set output to ""
	repeat with c in characters of str
		set charCode to id of c
		if c is "\"" then
			set output to output & "\\\""
		else if c is "\\" then
			set output to output & "\\\\"
		else if charCode is 8 then
			set output to output & "\\b"
		else if charCode is 9 then
			set output to output & "\\t"
		else if charCode is 10 then
			set output to output & "\\n"
		else if charCode is 12 then
			set output to output & "\\f"
		else if charCode is 13 then
			set output to output & "\\r"
		else if charCode < 32 then
			set output to output & "\\u00" & my toHexByte(charCode)
		else
			set output to output & c
		end if
	end repeat
	return output
end escapeJSON
