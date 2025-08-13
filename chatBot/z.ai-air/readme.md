# z.ai GLM-4.5-Air

in the responses below, the AI made at least five glaring mistakes, which means
it is too stupid to be useful

## prompt 1, 1m15s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 1m18s

BaseURL is string not slice

## prompt 3, 14s

Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`

## prompt 4, 1m8s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 5, 1m12s

support SegmentTimeline

## prompt 6, 27s

247:9: declared and not used: timeInSeconds

## prompt 7, 40s

247:9: declared and not used: currentTimeInSeconds

## prompt 8, 51s

you are not replacing `$RepresentationID$` here:

~~~
initURL := resolveRelativeURL(baseURL, template.Initialization)
segments = append(segments, initURL)
~~~

## prompt 9, 42s

`$Time$` is an integer

## prompt 10, 34s

205:5: declared and not used: timescale

## prompt 11, 1m3s

`$Time$` value should persist across S elements

## prompt 12, 38s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 13

301:6: replacePlaceholders redeclared in this block
