# gemini flash

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

six file pass

Please provide the GoLang script that parses an MPEG-DASH MPD file and extracts segment URLs.

The script should adhere to the following rules:

* **BaseURL Resolution:** Resolve `BaseURL` elements hierarchically (MPD, Period, AdaptationSet, Representation) against a starting `http://test.test/test.mpd`.
* **Segment List Handling:** Parse `SegmentList` elements, including `Initialization` (as a child element with `sourceURL`) and `SegmentURL` elements.
* **Segment Template Handling:**
    * Inherit `SegmentTemplate` from `AdaptationSet` to `Representation` if the Representation doesn't have its own.
    * Handle `initialization` attribute for `SegmentTemplate`.
    * **SegmentTimeline:** If `SegmentTimeline` is present, generate segments based on `S` elements, considering `t`, `d`, and `r` attributes. The `$Number$` placeholder should start from `SegmentTemplate@startNumber` (defaulting to 1 if missing).
    * **EndNumber:** If `SegmentTimeline` is missing but `endNumber` is present, generate segments from `startNumber` to `endNumber`.
    * **Calculated Count:** If both `SegmentTimeline` and `endNumber` are missing, but `duration` and `timescale` are present in `SegmentTemplate`, calculate the number of segments using `ceil(PeriodDurationInSeconds * timescale / duration)`. `SegmentTemplate@timescale` should default to `1` if missing.
* **DASH Identifier Expansion:**
    * Expand `$RepresentationID$` with the actual Representation ID.
    * Expand `$Time$` with the calculated time (if `SegmentTimeline` is used) or `0` otherwise.
    * Expand `$Number$` with the segment number.
    * Expand zero-padded `$Number%0xd$` placeholders (for `x` from 2 to 9, e.g., `$Number%03d$`). Ensure these padded expansions are performed *before* the generic `$Number$` expansion.
* **Segment Aggregation:** Append segments for the same `Representation ID` if it appears in multiple `Periods`.
* **Output:** The final output should be a JSON object mapping Representation IDs to an array of their resolved segment URLs.
* **Error Handling/Warnings:** Provide informative warnings for malformed URLs, unparseable durations/numbers, or insufficient attributes to extract segments.
