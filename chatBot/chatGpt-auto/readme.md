# chatGpt auto

## prompt 1, 19s

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 17s

support SegmentTimeline

## prompt 3, 13s

each BaseURL should build on the previous result, which starts with the MPD URL
previously given

## prompt 4, 13s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 5, 15s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 6, 15s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 7, 20s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 8, 18s

Default `timescale` to `1` if missing

## prompt 9, 22s

replace input like `$Number%08d$`

## prompt 10, 52s

startNumber missing is different from startNumber="0"
