# chatGpt 4o

https://chatgpt.com?model=gpt-4o

## prompt 1

Please return the full Go script that:

- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2

support SegmentTimeline

## prompt 3

`segmentCount >= 5` is limiting the count when that was not requested

## prompt 4

respect Period@BaseURL

## prompt 5

use only net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 6

respect SegmentTemplate@endNumber

## prompt 7

support SegmentList
