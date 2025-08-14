# chatGpt 4o

## prompt 1, 11s

Please return the full Go script that:
- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 8s

use only net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 3, 8s

respect Period@BaseURL

## prompt 4, 12s

replace `$Time$`

## prompt 5, 13s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 6, 9s

support SegmentList

## prompt 7, 16s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 8, 9s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 9, 13s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 10, 4s

Default `timescale` to `1` if missing

## prompt 11, 7s

Period should use its own duration if possible:

periodDurationSeconds := parseISODuration(mpd.MediaPresentationDuration)

## prompt 12, 5s

replace input like `$Number%08d$`

## prompt 13, 2s

20:2: undefined: re

## prompt 14, 20s

missing startNumber is different than startNumber="0"
