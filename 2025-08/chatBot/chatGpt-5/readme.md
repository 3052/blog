# chatGpt 5

## prompt 1, 1m8s

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 7s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 3, 5s

replace input like `$Number%08d$`
