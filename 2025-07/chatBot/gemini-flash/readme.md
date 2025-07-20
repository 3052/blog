# gemini flash

https://gemini.google.com

do not use Gemini canvas

1. Go language script, input is path to local DASH MPD file
2. output is map, key is Representation ID, value is segment URLs
3. marshal output with json.Marshal
4. if logging use standard error
5. standard library only
6. assume MPD URL is http://test.test/test.mpd
8. use net/url.Parse or net/url.URL.Parse to create URL
9. respect Period@BaseURL
10. if Representation is missing SegmentBase, SegmentList, SegmentTemplate,
   return Representation@BaseURL
11. if Representation spans Periods, append URLs
12. SegmentTemplate is a child of AdaptationSet or Representation
13. SegmentTemplate@timescale is 1 if missing
14. SegmentTemplate@startNumber is 1 if missing
15. replace $RepresentationID$ with Representation@id
16. $Number$ value should increase by 1 each iteration
17. SegmentTemplate@endNumber can exist. if so it defines the last segment
18. if no SegmentTemplate@endNumber use SegmentTimeline
19. if no SegmentTimeline use
   math.Ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
