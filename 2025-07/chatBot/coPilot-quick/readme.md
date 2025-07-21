# coPilot quick response

- <https://wikipedia.org/wiki/Microsoft_Copilot>
- https://copilot.microsoft.com

1. Go language script, user input is path to local DASH MPD file
2. output is map, key is Representation ID, value is segment URLs
3. format output with json.Marshal
4. if Representation is missing SegmentList, and Representation is missing
   SegmentTemplate and AdaptationSet is missing SegmentTemplate, return
   Representation@BaseURL
5. each S element will be used 1 + S@r times
6. SegmentTemplate@startNumber is 1 if missing
7. replace $RepresentationID$ with Representation@id
8. SegmentTemplate@timescale is 1 if missing
9. struct fields must include a tag unless the name matches
10. if Representation spans Periods, append URLs
13. $Number$ value should increase by 1 each iteration
14. $Time$ value should increase by S@d each iteration
16. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
17. if SegmentTemplate@endNumber exists, use to determine segments
18. if SegmentTimeline exists, use to determine segments
19. do not use net/url.URL.Path to construct a URL
20. if URL is relative, use net/url resolve to absolute with
   1. http://test.test/test.mpd
   2. MPD@BaseURL
   3. Period@BaseURL
