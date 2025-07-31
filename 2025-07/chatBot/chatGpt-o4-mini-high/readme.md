# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

Please show me the complete Go program (using only the standard library) that:

- Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
- Starts with `const defaultBase = "http://test.test/test.mpd"` and chains each
  `<BaseURL>` (MPD → Period → Representation) via
   `net/url.URL.ResolveReference`
- Outputs a JSON map from each `Representation.ID` to its ordered list of fully resolved segment URLs

---

- Supports both `<SegmentList>` (with `<Initialization>` and `<SegmentURL>`) and `<SegmentTemplate>` (handling `initialization`, a `<SegmentTimeline>` with `$Number$` or `$Time$`, numeric `startNumber`/`endNumber`, or—if no timeline/endNumber but both `duration` and `timescale` are present—computes `ceil(PeriodDurationInSeconds * timescale / duration)`)
- Substitutes `$RepresentationID$`, `$Number…$` (with optional zero‐padding), and `$Time…$` placeholders
- Inherits missing templates and defaults `timescale` to 1
- Appends segments across multiple `<Period>`s for the same representation ID
- Parses ISO 8601 durations like `PT2H13M19.040S`
- If a Representation only has its own `<BaseURL>`, returns that URL as the sole segment
