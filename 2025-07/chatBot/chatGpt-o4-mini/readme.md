# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

three file pass

Please show me the complete Go program using only the standard library that:

* Reads an MPEG‑DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
* Starts with `const defaultBase = "http://test.test/test.mpd"` and chains each `<BaseURL>` (MPD → Period → AdaptationSet → Representation) via `net/url.URL.ResolveReference`
* Supports both `<SegmentList>` (with `<Initialization>` and `<SegmentURL>`) and `<SegmentTemplate>` (with `initialization`, segment timelines, numeric `startNumber`/`endNumber`, or, if both `duration` and `timescale` are present but no timeline/`endNumber`, computes `ceil(PeriodDurationSeconds * timescale / duration)`)
* Substitutes `$RepresentationID$`, `$Number[…]$`, and `$Time[…]$` placeholders
* Inherits missing templates and defaults `timescale` to 1
* Appends segments across multiple `<Period>`s for the same representation ID
* Parses ISO8601 durations like `PT2H13M19.040S`
* Outputs a JSON map from each `Representation@id` to its ordered list of fully resolved segment URLs

---

- Supports:
  - `<SegmentTimeline>` with proper handling of `@t`, `@d`, `@r`
  - `<SegmentList>` and `<Initialization>` elements
