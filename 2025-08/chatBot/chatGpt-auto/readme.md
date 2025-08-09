# chatGpt auto

https://chatgpt.com?model=auto

## prompt 1, 13s

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 18s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 3, 8s

replace `$RepresentationID$`

## prompt 4, 11s

replace `$Time$`

## prompt 5, 14s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 6, 14s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 7, 5s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 8, 19s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 9, 18s

each BaseURL value should build upon the previous result

## prompt 10, 15s

only use net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 11, 10s

`$Time$` value should persist across S elements

## prompt 12, 25s

support SegmentList

## prompt 13, 5s

Default `timescale` to `1` if missing

## prompt 14, 6s

replace input like `$Number%08d$`
