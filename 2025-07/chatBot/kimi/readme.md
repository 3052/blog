# kimi

https://kimi.com

Give me the complete Go program that
- reads a local `.mpd` file (first CLI arg),
- starts from the original MPD URL `http://test.test/test.mpd`,
- resolves the BaseURL hierarchy (MPD → Period → AdaptationSet → Representation) using **only** `net/url.Parse` + `ResolveReference`,
- expands `SegmentTemplate` with `SegmentTimeline`, `$RepresentationID$`, `$Bandwidth$`, `$Number$`, `$Time$`, `%0xd` padding, and honours `@endNumber` when present,
- outputs pure JSON `{ "RepresentationID": [ "absolute_url", … ] }`.

provide prompt in markdown I can give you in the future to return this script

---

1. Go language script, input is local DASH MPD file
2. script is called as "dash input.mpd" or similar
3. output is map, key is Representation ID, value is segment URLs
4. for each segment, use net/url
    1. start with http://test.test/test.mpd
    2. resolve MPD@BaseURL
    3. resolve Period@BaseURL
    4. if Representation@BaseURL exists, resolve and return
    5. else resolve segment URL and return
5. format output with json.Marshal
6. SegmentTemplate is a child of AdaptationSet or Representation
7. standard library only
8. if SegmentTemplate@endNumber exists, use to determine segments
9. SegmentTemplate@startNumber is 1 if missing
10. each S element will be used 1 + S@r times
11. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
12. SegmentTemplate@timescale is 1 if missing
13. if Representation spans Periods, append URLs
14. if SegmentTimeline exists
    1. declare variable startTime
    2. replace `$Time$` with startTime
    3. replace `$Number$` with SegmentTemplate@startNumber
    4. increment startTime by S@d
    5. increment SegmentTemplate@startNumber by 1
15. replace `$RepresentationID$` in SegmentTemplate@media
16. replace `$Number%02d$` and similar once in SegmentTemplate@media
17. replace `$Number$` in SegmentTemplate@media
18. handle all errors
19. no duplicate URLs

---

13. use only net/url.Parse to construct URLs
14. use only net/url.URL.ResolveReference to resolve URLs
