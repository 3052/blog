# gemini flash

## prompt 1, 7s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 7s

support SegmentTimeline

## prompt 3, 7s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 4, 7s

use net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 5, 8s

respect Period@BaseURL

## prompt 6, 8s

replace `$Time$`

## prompt 7, 10s

support SegmentTemplate@endNumber

## prompt 8, 10s

with SegmentList include Initialization@sourceURL if it exists

## prompt 9, 10s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 10, 11s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 11, 16s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 12, 13s

Default `timescale` to `1` if missing

## prompt 13, 14s

replace input like `$Number%08d$`

## prompt 14, 14s

default SegmentTemplate@startNumber to 1 if missing
