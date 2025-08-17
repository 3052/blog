# venice

You've exceeded the number of Chat requests you can make today. Please come
back again tomorrow or upgrade to Venice Pro to obtain higher daily limits

## prompt 1, 18s

Please return the full Go script that:
- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 23s

support SegmentTemplate

## prompt 3, 35s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 4, 33s

any logs should go to standard error

## prompt 5, 27s

do not use `interface{}` for any reason unless you provide a good explanation

## prompt 6, 30s

SegmentTemplate@numberofSegments is not a valid attribute

## prompt 7, 35s

support SegmentTimeline

## prompt 8, 44s

161:38: undefined: segmentTemplate

## prompt 9, 43s

SegmentTemplate@startNumber is optional

## prompt 10, 41s

SegmentTimeline is a child of SegmentTemplate

## prompt 11, 45s

~~~
139:97: cannot use &segmentURLs (value of type **[]string) as *[]string value
in argument to handleSegmentTimeline
~~~

## prompt 12, 44s

SegmentTemplate@duration is optional

## prompt 13

S@t is optional
