# coPilot think deeper

https://copilot.microsoft.com

1. Go language script, input is local DASH MPD file
2. output is map, key is Representation ID, value is segment URLs
3. format output with json.Marshal
4. using net/url resolve MPD@BaseURL from http://test.test/test.mpd
5. resolve Period@BaseURL from result
6. resolve segment URL from result
7. SegmentTemplate is a child of AdaptationSet or Representation
8. if SegmentTimeline exists, use to determine segments
9. replace `$RepresentationID$` in SegmentTemplate@media
10. if SegmentTemplate@endNumber exists, use to determine segments
11. `$Time$` value should increase by S@d each iteration
12. do not use net/url.URL.Path to construct a URL
13. replace `$Number$` in SegmentTemplate@media

---

4. Representation is a child of AdaptationSet
6. if Representation@BaseURL exists, treat it as a segment URL
7. each S element will be used 1 + S@r times
8. SegmentTemplate@startNumber is 1 if missing
9. SegmentTemplate@timescale is 1 if missing
10. no duplicate URLs
11. if Representation spans Periods, append URLs
14. replace `$Number%` in SegmentTemplate@media
15. `$Number$` value should increase by 1 each iteration
17. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
