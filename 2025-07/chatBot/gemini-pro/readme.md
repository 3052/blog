# gemini pro

provide markdown prompt I can give you in the future to return this script

https://gemini.google.com

six file pass

Please provide a complete, self-contained Go script in a single `main.go` file
that performs the following actions:

## 📜 Script Requirements

### 1. Execution
* The script must parse a local MPEG-DASH MPD file path passed as a command-line argument: `go run main.go <path_to_mpd_file>`.

---
### 2. URL Resolution
* It must use the hardcoded URL `http://test.test/test.mpd` as the top-level base for resolving all relative paths.
* **Resolution Hierarchy**: The script must correctly resolve relative URLs by respecting the hierarchy of `<BaseURL>` elements. A `<BaseURL>` at a lower level (e.g., `Representation`) is resolved relative to the base URL established by the level above it (e.g., `Period`).

---
### 3. Inheritance
* A `Representation` must inherit `<SegmentTemplate>`, `<SegmentList>`, and `<Initialization>` elements from its parent `AdaptationSet`.
* Any element or attribute defined directly on a `Representation` must override the corresponding one inherited from the `AdaptationSet`.

---
### 4. Initialization Segment
* The script must find the initialization segment URL by checking for its definition in the following order of precedence:
    1.  An `<Initialization>` element that is a direct child of the `Representation`.
    2.  An `<Initialization>` element that is a child of the effective `<SegmentList>`.
    3.  The `initialization` attribute on the effective `<SegmentTemplate>`.
* **An initialization segment must be generated and prepended for every period in which its representation appears.** The final list of URLs for a representation will contain all of its initialization and media segments from all periods.

---
### 5. Segment Generation
* The script must generate the list of media segments using the first available method from the following list of precedence:
    1.  **Primary (`<SegmentTimeline>`)**: Iterate through its `<S>` elements, correctly handling `t`, `d`, and `r` attributes.
    2.  **Secondary (`<SegmentList>`)**: Use the effective `<SegmentList>` and create a URL for each of its `<SegmentURL>` children.
    3.  **Tertiary (Number-based Template)**: Use `startNumber` and `endNumber` from the effective `<SegmentTemplate>`.
    4.  **Quaternary (Duration-based Template)**: Use the `duration` and `timescale` from the effective `<SegmentTemplate>` along with the `Period@duration` (or `MPD@mediaPresentationDuration`) to calculate the segment count.
    5.  **Final Fallback (`<BaseURL>` List)**: Treat any `<BaseURL>` elements that are direct children of the `Representation` as a literal list of segment URLs.
* **Default Values**:
    * If `SegmentTemplate@timescale` is omitted, it must default to `1`.
    * If `SegmentTemplate@startNumber` is omitted, it must default to `1`.

---
### 6. Placeholder Substitution
* The script must correctly substitute `$RepresentationID$`, `$Number$`, and `$Time$` placeholders when constructing final segment URLs.
* Substitution must support `printf`-style formatting, such as `$Number%05d$`.

---
### 7. Multi-Period Representations
* If the same `Representation@id` appears in multiple `<Period>` elements, the script must append the segments from subsequent periods to the list for that representation.
* The segment numbering for a representation must **always** be determined by the attributes (`startNumber`, etc.) within its **current** `<Period>`. It should **not** create a continuous segment count across multiple periods.

---
### 8. Output
* The script must print a single, indented JSON object to standard output.
* This JSON object should map each `Representation@id` string to a flat list of all its resolved segment URL strings from all periods: `{ "rep_id": ["period1_init_url", "period1_segment_1_url", ..., "period2_init_url", "period2_segment_1_url", ...] }`.
