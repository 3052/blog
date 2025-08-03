# chatGpt o3

https://chatgpt.com?model=o3

## prompt 1, 42 seconds

return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- starts BaseURL resolution with the absolute URL `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 27 seconds

BaseURL is string not slice

## prompt 3, 42 seconds

support SegmentTimeline

## prompt 4, 34 seconds

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 5, 50 seconds

replace input like `$Number%08d$`
