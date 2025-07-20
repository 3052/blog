# coPilot quick response

- <https://wikipedia.org/wiki/Microsoft_Copilot>
- https://copilot.microsoft.com

1. Go language script, input is path to local DASH MPD file
2. output is JSON object, key is Representation ID, value is segment URLs
3. use MPD URL http://test.test/test.mpd to resolve relative URLs
4. SegmentTemplate is a child of AdaptationSet or Representation
5. if Representation is missing SegmentList and SegmentTemplate, return
   Representation@BaseURL
6. if SegmentTimeline exists use to determine segments
7. use net/url.Parse or net/url.URL.Parse to create URLs

---

8. all errors should be fatal
9. if logging use standard error
10. standard library only
11. respect Period@BaseURL
12. if Representation spans Periods, append URLs
13. SegmentTemplate@timescale is 1 if missing
14. SegmentTemplate@startNumber is 1 if missing
15. replace $RepresentationID$ with Representation@id
16. $Number$ value should increase by 1 each iteration
17. SegmentTemplate@endNumber can exist. if so it defines the last segment
18. if no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
