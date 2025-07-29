# gemini pro

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

two file pass

Please provide a complete, self-contained Go script in a single `main.go` file that performs the following actions:

## 📜 Script Requirements

### 1. Execution
* The script must parse a local MPEG-DASH MPD file path passed as a command-line argument: `go run main.go <path_to_mpd_file>`.

---
### 2. URL Resolution
* It must use the hardcoded URL `http://test.test/test.mpd` as the top-level base for resolving all relative paths.
* **Resolution Hierarchy**: The script must correctly resolve relative URLs by respecting the following hierarchy of `<BaseURL>` elements, from highest to lowest scope:
    1.  `<Period>`
    2.  `<Representation>`
* A `<BaseURL>` at a lower level (e.g., `Representation`) is resolved relative to the base URL established by the level above it (e.g., `Period`). All relative paths within that scope are then resolved against this new, more specific base URL.

---
### 3. Inheritance
* A `Representation` must inherit `<SegmentTemplate>`, `<SegmentList>`, and `<Initialization>` elements from its parent `AdaptationSet`.
* Any element or attribute defined directly on a `Representation` must override the corresponding one inherited from the `AdaptationSet`.

---
### 4. Initialization Segment
* The script must find the initialization segment URL by checking for its definition in the following order of precedence (from highest to lowest):
    1.  An `<Initialization>` element that is a direct child of the `Representation`.
    2.  An `<Initialization>` element that is a child of the effective `<SegmentList>`.
    3.  The `initialization` attribute on the effective `<SegmentTemplate>`.
* The resulting initialization segment URL must be the first element in the final URL list for that representation.

---
### 5. Segment Generation
* The script must generate the list of media segments using the first available method from the following list of precedence:
    1.  **Primary (`<SegmentTimeline>`)**: If the effective `<SegmentTemplate>` contains a `<SegmentTimeline>`, iterate through its `<S>` elements, correctly handling `t` (time), `d` (duration), and `r` (repeat) attributes.
    2.  **Secondary (`<SegmentList>`)**: If no `<SegmentTimeline>` is found, use the effective `<SegmentList>` and create a URL for each of its `<SegmentURL>` children.
    3.  **Tertiary (Duration-based Template)**: If the above are missing, use the effective `<SegmentTemplate>`'s `duration` and `timescale` attributes to calculate the total number of segments based on the `MPD@mediaPresentationDuration`.
    4.  **Final Fallback (`<BaseURL>` List)**: If no other segment information exists, treat any `<BaseURL>` elements that are direct children of the `Representation` as a literal list of segment URLs.
* **URL Templates**: The script must correctly substitute the `$RepresentationID$`, `$Number$`, and `$Time$` placeholders when constructing the final segment URLs from a `<SegmentTemplate>`.

---
### 6. Output
* The script must print a single, indented JSON object to standard output.
* This JSON object should map each `Representation@id` string to a list of fully resolved segment URL strings: `{ "rep_id": ["init_url", "segment_1_url", ...] }`.

---

- Resolves `<BaseURL>` elements hierarchically: MPD → Period → AdaptationSet → Representation
- Handles:
  - `$RepresentationID$`, `$Number$`, `$Time$` substitutions, including formatted patterns like `$Number%05d$`
  - `startNumber` and `endNumber`
  - `$Time$` using `<SegmentTimeline>` and `@r` repetitions (each `<S>` repeated `1 + r` times)
  - If both `SegmentTimeline` and `endNumber` are missing, and `duration` + `timescale` are present, calculate number of segments as:
    `ceil(PeriodDurationInSeconds * timescale / duration)`
  - Defaults `timescale` to `1` if omitted
- Appends segments for the same `Representation@id` if it appears in multiple `<Period>` elements
- Uses **only** `net/url.URL.ResolveReference` for URL resolving (no other logic)
- Prefers `Period@duration` over `MPD@mediaPresentationDuration` for segment count calculations
