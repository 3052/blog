# gemini flash

## prompt 1, 7s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 4s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 3, 7s

give me the script

## prompt 4, 6s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 5, 6s

you are very obviously wrong, as is demonstrated in ISO/IEC 23009-1:2022, where
SegmentTemplate@endNumber is defined

## prompt 6, 8s

how many times do I need to say give me the script before you start giving me
the script in responses?

## prompt 7, 6s

SegmentList can be a child of Representation

## prompt 8, 18s

unless I say otherwise, when I give a prompt that means I want an updated
script or an updated snippet, why is that hard for you to understand?

## prompt 9, 8s

with SegmentList, include Initialization@sourceURL if it exists

## prompt 10, 9s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 11, 3s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 12, 8s

you are making something very simple sound difficult, literally all you need to
do is something like this:

~~~
segmentURLs["hello"] = append(segmentURLs["hello"], "world")
~~~

and move the initial declaration if needed

## prompt 13, 9s

it seems SegmentTemplate@startNumber is not being respected

## prompt 14, 11s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 15, 10s

if you generate logs they should print to standard error

## prompt 16, 10s

Default `timescale` to `1` if missing

## prompt 17, 13s

replace input like `$Number%08d$`

## prompt 18, 11s

support duration like `PT1H53M46.040S`

## prompt 19, 11s

119:6: declared and not used: err

## prompt 20, 11s

SegmentTemplate@startNumber should be `1` if missing

## prompt 21, 12s

missing startNumber is different than startNumber="0"

## prompt 22, 12s

`$Number$` value should persist through S elements

## prompt 23, 13s

your comment has nothing to do with my last prompt
