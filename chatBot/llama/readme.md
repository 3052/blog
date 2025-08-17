# llama

this AI is obviously too stupid to be useful

## prompt 1, 16s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 10s

standard library only

## prompt 3, 15s

160:22: url.Parse undefined (type string has no field or method Parse)

## prompt 4, 6s

full script

## prompt 5, 8s

163:22: url.Parse undefined (type string has no field or method Parse)

## prompt 6, 7s

17:15: undefined: neturl
