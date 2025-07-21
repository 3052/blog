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
7. use net/url.Parse or net/url.URL.Parse to create URLs
8. if SegmentTimeline exists, use to determine segments
9. if SegmentTemplate@endNumber exists, use to determine segments
10. each S element will be used 1 + S@r times
11. SegmentTemplate@startNumber is 1 if missing
12. replace $RepresentationID$ with Representation@id
13. respect Period@BaseURL
14. if no SegmentTemplate@endNumber and no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
15. SegmentTemplate@timescale is 1 if missing
16. struct fields must include a tag unless the name matches
17. $Number$ value should increase by 1 each iteration
18. standard library only
19. if Representation spans Periods, append URLs

---

8. all errors should be fatal
9. if logging use standard error
