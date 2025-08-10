# deepSeek

as can be seen, this model is too stupid to be useful

## prompt 1, 57s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 59s

SegmentTemplate can be child of Representation or AdaptationSet

## prompt 3, 1m19s

support SegmentTimeline

## prompt 4, 1m10s

use net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 5, 23s

`$Time$` value should persist across S elements

## prompt 6, 16s

`$Time$` should be incremented after replacement not before

## prompt 7, 1m5s

SegmentTemplate@endNumber can be used instead of SegmentTimeline

## prompt 8, 58s

support SegmentList

## prompt 9, 58s

with SegmentList, include Initialization@sourceURL if it exists

## prompt 10, 31s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 11, 44s

Append segments for the same Representation ID if it appears in multiple
Periods

## prompt 12, 25s

SegmentTemplate@media can include `$Number$` with SegmentTimeline

## prompt 13, 39s

If both `SegmentTimeline` and `endNumber` are missing, but `duration` and
`timescale` are present, calculate the number of segments using
`ceil(PeriodDurationInSeconds * timescale / duration)`

## prompt 14, 1m28s

full script

## prompt 15, 23s

Default `timescale` to `1` if missing

## prompt 16, 1m49s

do not ignore errors

## prompt 17, 35s

I just said do not ignore errors, and you are continuing to do so:

~~~
d, err := time.ParseDuration(strings.ToLower(dur[1:len(dur)-1] + "s"))
if err != nil {
   return 0
}
~~~

## prompt 18, 1m4s

140:33: assignment mismatch: 1 variable but processSegmentTemplate returns 2
values

## prompt 19, 1m57s

full script

## prompt 20, 19s

this is wrong because it still includes `t`:

goDur := strings.ToLower(dur[1:])

## prompt 21, 24s

periodDur should be in seconds:

~~~
float64(periodDur*time.Duration(timescale))
~~~

## prompt 22, 28s

405:109: syntax error: unexpected newline in argument list; possibly missing
comma or )

## prompt 23, 11s

your parenthesis are not balanced:

~~~
durationInTimescale := int(math.Round(float64(periodDur.Seconds() * float64(timescale)))
~~~
