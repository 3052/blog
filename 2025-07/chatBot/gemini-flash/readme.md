# gemini flash

https://gemini.google.com

no canvas

1. Go language script, input is path to local DASH MPD file
2. output is JSON object, key is Representation ID, value is segment URLs
3. if logging use standard error
4. standard library only
5. assume MPD URL is http://test.test/test.mpd
6. use net/url.Parse or net/url.URL.Parse to create URL
8. Representation with only BaseURL means one segment
9. Representations can be split across Periods
10. SegmentTemplate is a child of AdaptationSet or Representation
11. SegmentTemplate@timescale is 1 if missing
12. SegmentTemplate@startNumber is 1 if missing
13. $Number$ value should increase by 1 each iteration
14. SegmentTemplate@endNumber can exist. if so it defines the last segment
15. if no SegmentTemplate@endNumber use SegmentTimeline
16. if no SegmentTimeline use
   ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
