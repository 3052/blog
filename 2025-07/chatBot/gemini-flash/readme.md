# gemini flash

https://gemini.google.com

do not use Gemini canvas

1. Go language script, input is path to local DASH MPD file
2. output is JSON object, key is Representation ID, value is segment URLs
3. if logging use standard error
4. standard library only
5. assume MPD URL is http://test.test/test.mpd
7. use net/url.Parse or net/url.URL.Parse to create URL
8. use math.Ceil if needed
9. if Representation is missing SegmentBase, SegmentList, SegmentTemplate, return
   Representation@BaseURL
10. if Representation spans Periods, append URLs
11. SegmentTemplate is a child of AdaptationSet or Representation
12. SegmentTemplate@timescale is 1 if missing
13. SegmentTemplate@startNumber is 1 if missing
14. replace $RepresentationID$ with Representation@id
15. $Number$ value should increase by 1 each iteration
16. SegmentTemplate@endNumber can exist. if so it defines the last segment
17. if no SegmentTemplate@endNumber use SegmentTimeline
18. if no SegmentTimeline use
   ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
