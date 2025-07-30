# chatGpt

provide markdown prompt I can give you in the future to return this script

https://chatgpt.com

one file pass

**Task:** Return a complete Go program (using only the standard library) that:

1. Reads an MPEG-DASH MPD XML file path from the CLI:  
   go run main.go <mpd_file_path>
2. Uses http://test.test/test.mpd as the base URL for resolving all relative URLs.
3. Supports both <SegmentList> and <SegmentTemplate> (at AdaptationSet and Representation levels), with inheritance.
4. Performs placeholder substitution for:  
   - $RepresentationID[...]$  
   - $Number[...]$  
   - $Time[...]$  
   including formatted forms like $Number%05d$.
5. Persists the $Time$ value continuously across timeline entries.
6. Respects endNumber in numeric templates.
7. Outputs a JSON mapping of each Representation@id to its ordered list of fully‑resolved segment URLs (initialization first).

---

- Supports:
  - `<SegmentTimeline>` with proper handling of `@t`, `@d`, `@r`
  - `<SegmentList>` and `<Initialization>` elements
  - `<BaseURL>` hierarchy: MPD → Period → AdaptationSet → Representation
  - Fallback to `BaseURL` segment if no other segment info is present
  - Always uses `url.URL.ResolveReference` for URL resolution
  - Appends segments across multiple `<Period>`s for the same `Representation@id`
  - Defaults `timescale=1` if not present
  - If both `SegmentTimeline` and `endNumber` are missing, and both `duration` and `timescale` are present, calculates number of segments as:
    ```
    ceil(PeriodDurationInSeconds * timescale / duration)
    ```
