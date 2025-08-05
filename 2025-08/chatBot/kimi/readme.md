# kimi

<https://wikipedia.org/wiki/Moonshot_AI>

despite repeatedly correcting the AI on URL logic, it is still making mistakes
like adding file extension:

~~~
seg, _ := url.Parse(repID + ".mp4")
~~~

## prompt 1, 3m32s

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2, 44s

137:6: declared and not used: timescale

## prompt 3, 45s

BaseURL is string not slice

## prompt 4, 2m4s

full script

## prompt 5, 2m56s

support SegmentTimeline

## prompt 6, 36s

109:6: declared and not used: timescale

## prompt 7, 1m9s

192:7: declared and not used: mediaTemplate

## prompt 8, 12s

replace `$RepresentationID$`

## prompt 9, 1m57s

use only net/url.Parse to create URLs, no other package or logic

## prompt 10, 20s

replace `$RepresentationID$`

## prompt 11, 1m59s

full script

## prompt 12, 25s

this code has absolutely nothing to do with the DASH spec

~~~
q := tmplURL.Query()
q.Set("$Number$", strconv.Itoa(segNum))
q.Set("$Time$", strconv.FormatInt(t, 10))
q.Set("$Bandwidth$", "0")
u := *tmplURL
u.RawQuery = q.Encode()
~~~

## prompt 13, 17s

154:4: declared and not used: tmplURL

## prompt 14, 40s

again:

declared and not used: tmplURL

## prompt 15, 49s

include SegmentTemplate@initialization if it exists

## prompt 16, 1m3s

support SegmentTemplate@endNumber

## prompt 17, 2m23s

full script

## prompt 18, 11s

~~~
118:47: list.Duration undefined (type *SegmentList has no field or method
Duration)
~~~

## prompt 19, 1m9s

with SegmentList support SegmentURL@media

## prompt 20, 1m11s

with SegmentList include Initialization@sourceURL if it exists

## prompt 21, 1m17s

When a representation has only BaseURL and no segment information, use the
already-resolved baseURL directly

## prompt 22, 2m30s

again, use only net/url.Parse to create URLs, no other package or logic

## prompt 23, 2m19s

third request, use only net/url.Parse to create URLs, no other package or logic

## prompt 24, 2m18s

use only net/url.URL.ResolveReference to resolve URLs, no other package or logic

## prompt 25, 1m11s

other than the below, do not create or update URLs at all

- net/url.Parse
- net/url.URL.ResolveReference
- replacements for SegmentTemplate

## prompt 26, 35s

I was extremely clear with you, yet you are still updating URL with
`base.Path += "/"`
