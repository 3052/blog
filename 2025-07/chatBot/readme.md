# chatBot

## gemini flash

https://gemini.google.com

1. Go language script, input is path to local DASH MPD file
2. output is JSON object, key is Representation ID, value is segment URLs
3. all errors should be fatal
4. if logging use standard error
5. standard library only
6. assume MPD URL is http://test.test/test.mpd
8. use net/url.Parse or net/url.URL.Parse to create URL
9. respect Period@BaseURL
10. if Representation is missing SegmentList and SegmentTemplate, return
   Representation@BaseURL
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

I understand your frustration, and I apologize that I cannot fulfill your
request for a complete, production-ready Go script that fully implements all
the complex logic for parsing DASH MPD files and generating segment URLs.

## gemini deep research

https://gemini.google.com

does not return an actual script

---

## claude

https://claude.ai

piece of shit - even the paid plan you hit your limit in 10 minutes

## coPilot

- <https://wikipedia.org/wiki/Microsoft_Copilot>
- https://copilot.microsoft.com

can only do a single file

## deepSeek

- https://chat.deepseek.com
- https://wikipedia.org/wiki/DeepSeek

free version will not let me attach the six test files, no paid version

## gemini pro

https://gemini.google.com

You’ve reached your limit on 2.5 Pro until Jul 19, 3:12 PM. Try Google AI Pro
for higher limits.

## kimi

https://kimi.com

free version will not let me attach the six test files

## moonshot

API is trash:

- https://platform.moonshot.ai/docs/api/caching
- https://platform.moonshot.ai/docs/api/files

## grok

https://grok.com

Apologies, your request is currently too long for our circuits to process.
Please, try a shorter version, won't you?
