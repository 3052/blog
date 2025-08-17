# mistral

web client is poorly designed, such that once you get past 10 prompts the
interface lags terribly

## prompt 1, 10s

return the full Go script that:
- Parses a local MPEG-DASH MPD XML file from a CLI argument: `go run main.go <mpd_file_path>`
- Uses base URL: `http://test.test/test.mpd`
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs (initialization segment first, if present)

## prompt 2, 3s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 3, 2s

support SegmentTimeline

## prompt 4, 6s

replace `$RepresentationID$` in all URLs

## prompt 5, 5s

respect Period@BaseURL

## prompt 6, 42s

replace `$Time$`

## prompt 7, 4s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 8, 4s

~~~
name "Initialization" in tag of main.SegmentList.Initialization conflicts with
name "SegmentURL" in *main.SegmentURL.XMLName
~~~

## prompt 9, 5s

~~~
name "Initialization" in tag of main.SegmentList.Initialization conflicts with
name "SegmentURL" in *main.SegmentURL.XMLName
~~~

## prompt 10, 9s

~~~
name "Initialization" in tag of struct { Initialization *main.SegmentURL
"xml:\"Initialization,omitempty\""; SegmentURLs []main.SegmentURL
"xml:\"SegmentURL\"" }.Initialization conflicts with name "SegmentURL" in
*main.SegmentURL.XMLName
~~~

## prompt 11, 6s

~~~
Error parsing MPD: expected element type <SegmentURL> but have <Initialization>
~~~

## prompt 12, 5s

~~~
Error parsing MPD: expected element type <SegmentURL> but have <Initialization>
~~~

## prompt 13, 54s

Initialization child of SegmentList should have sourceURL attribute not media

## prompt 14, 49s

with SegmentList, include Representation@BaseURL if it exists

## prompt 15, 49s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 16, 48s

this logic is obviously wrong because it ignores the previous instruction of
SegmentTemplate can be child of Representation or AdaptationSet:

~~~
if representation.SegmentTemplate == nil && representation.SegmentList == nil {
~~~

## prompt 17, 1m19s

133:5: syntax error: unexpected keyword else, expected }
