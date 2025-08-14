# gpt-oss-20b-high

## prompt 1, 2m37s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 1m5s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 3, 1m25s

replace `$RepresentationID$` in any URL that includes placeholders

## prompt 4, 1m30s

replace `$Time$`

## prompt 5, 1m23s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 6, 1m

do not assume SegmentTemplate exists

## prompt 7, 59s

with SegmentList, include Initialization@sourceURL if it exists. note that
Initialization as a child of SegmentList is different than initialization
attribute of SegmentTemplate

## prompt 8, 1m15s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 9, 59s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 10, 1m44s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 11, 1m5s

Default `timescale` to `1` if missing

## prompt 12, 1m31s

replace input like `$Number%08d$`

## prompt 13, 1m31s

missing startNumber is different than startNumber="0"
