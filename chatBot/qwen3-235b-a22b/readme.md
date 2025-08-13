# qwen3-235b-a22b

https://app.fireworks.ai/models/fireworks/qwen3-coder-480b-a35b-instruct

this model at least is worthless, as a single prompt is taking more than five
minutes see the final prompt below

## prompt 1, 42s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 32s

do as I ask, BaseURL field is string not anything else

## prompt 3, 33s

158:22: undefined: as

## prompt 4, 36s

use net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 5, 19s

93:2: declared and not used: resolve

## prompt 6, 27s

115:24: undefined: rep

## prompt 7, 30s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 8, 37s

~~~
176:25: rep.SegmentList.Initialization undefined (type *SegmentList has no
field or method Initialization)
~~~

## prompt 9, 33s

replace `$RepresentationID$`

## prompt 10, 34s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 11, 51s

with SegmentList, include Initialization@sourceURL if it exists

## prompt 12, 54s

it seems you have no idea what you are doing. Initialization is a child element
of SegmentList, and sourceURL is an attribute of Initialization. note that
SegmentTemplate has a completely different layout, so do not assume they are
similar at all

## prompt 13, 49s

no, again you are completely wrong. SegmentURL has a media attribute only,
nothing else

## prompt 14, 4m10s

full script as single code block

## prompt 15, 2m50s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 16, 2m44s

support SegmentTemplate@startNumber

## prompt 17, 4m6s

this whole block is stupid, because again its not respecting
SegmentTemplate@startNumber, and its using the `$Time$` value to replace
`$Number$`

~~~
var time int64
for _, s := range segmentTemplate.SegmentTimeline.S {
   repeat := 1
   if s.R > 0 {
      repeat = int(s.R) + 1
   }
   for i := 0; i < repeat; i++ {
      media := strings.ReplaceAll(mediaTemplate, "$Time$", strconv.FormatInt(time, 10))
      media = strings.ReplaceAll(media, "$Number$", strconv.FormatInt(time, 10))
~~~

## prompt 18, 5m7s

`$Time$` value should persist across S elements
