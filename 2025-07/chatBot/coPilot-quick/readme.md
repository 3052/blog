# coPilot quick response

- <https://wikipedia.org/wiki/Microsoft_Copilot>
- https://copilot.microsoft.com

1. Go language script, user input is path to local DASH MPD file
2. output is map, key is Representation ID, value is segment URLs
3. format output with json.Marshal
4. Representation is a child of AdaptationSet
5. SegmentTemplate is a child of AdaptationSet or Representation
6. if Representation@BaseURL exists, treat it as a segment URL
7. each S element will be used 1 + S@r times
8. SegmentTemplate@startNumber is 1 if missing
9. SegmentTemplate@timescale is 1 if missing
10. no duplicate URLs
11. if Representation spans Periods, append URLs
12. replace `$RepresentationID$` in SegmentTemplate@media
13. replace `$Number$` in SegmentTemplate@media
14. replace `$Number%` in SegmentTemplate@media
15. `$Number$` value should increase by 1 each iteration
16. `$Time$` value should increase by S@d each iteration
17. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
18. if SegmentTemplate@endNumber exists, use to determine segments
19. if SegmentTimeline exists, use to determine segments
20. do not use net/url.URL.Path to construct a URL
21. if URL is relative, use net/url resolve to absolute with
    1. http://test.test/test.mpd
    2. MPD@BaseURL
    3. Period@BaseURL
