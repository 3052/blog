# chatGpt 4o

## prompt 1, 11s

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)
