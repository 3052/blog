# coPilot quick response

- <https://wikipedia.org/wiki/Microsoft_Copilot>
- https://copilot.microsoft.com

1. Go language script, user input is path to local DASH MPD file
2. output is map, key is Representation ID, value is segment URLs
3. format output with json.Marshal
4. use MPD URL http://test.test/test.mpd to resolve relative URLs
5. SegmentTemplate is a child of AdaptationSet or Representation
6. if Representation is missing SegmentList and SegmentTemplate, return
   Representation@BaseURL
7. if SegmentTimeline exists, use to determine segments
8. if SegmentTemplate@endNumber exists, use to determine segments
9. each S element will be used 1 + S@r times
10. SegmentTemplate@startNumber is 1 if missing
11. replace $RepresentationID$ with Representation@id
12. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
13. SegmentTemplate@timescale is 1 if missing
14. struct fields must include a tag unless the name matches
15. $Number$ value should increase by 1 each iteration
16. standard library only
17. if Representation spans Periods, append URLs
18. respect MPD@BaseURL
19. respect Period@BaseURL
20. do not use net/url.URL.Path to construct a URL
21. use net/url.Parse or net/url.URL.Parse to construct a URL

---

8. all errors should be fatal
9. if logging use standard error
