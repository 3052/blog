# chatGpt

https://chatgpt.com?model=o4-mini-high

as can be clearly seen below, this model is too stupid to be useful

## prompt 1, 35 seconds

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 19 seconds

BaseURL is string not slice

## prompt 3, 26 seconds

previous response failed to generate a script

## prompt 4, 20 seconds

scripts ended unexpectedly at line 245

## prompt 5, 15 seconds

scripts ended unexpectedly at line 213

## prompt 6, 14 seconds

scripts ended unexpectedly at line 230

## prompt 7, 13 seconds

scripts ended unexpectedly at line 191

## prompt 8, 29 seconds

invalid newline at line 154

## prompt 9, 14 seconds

scripts ended unexpectedly at line 190

## prompt 10, 28 seconds

support SegmentTimeline

## prompt 11, 23 seconds

each BaseURL should build from the previous result

## prompt 12, 20 seconds

include SegmentTemplate@initialization if it exists

## prompt 13, 22 seconds

include SegmentList.Initialization@sourceURL if it exists

## prompt 14, 10 seconds

invalid string syntax at line 243

## prompt 15, 20 seconds

`http://test.test/test.mpd` is the initial BaseURL regardless of MPD@BaseURL

## prompt 16, 25 seconds

MPD@BaseURL should not be ignored, it should be resolved against any previous
BaseURL as already instructed

## prompt 17, 13 seconds

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 18, 36 seconds

218:17: undefined: sl

## prompt 19, 9 seconds

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 20, 11 seconds

assume SegmentTemplate@media can always include `$Number$`
