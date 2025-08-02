# chatGpt

provide prompt I can give you to return this script

https://chatgpt.com

six file pass

Write a single, fully-compilable Go program named main.go that uses only the
standard library and satisfies *all* of the requirements below.

### CLI & basic behaviour
* Usage: `go run main.go <mpd_file_path>`
* Read the local MPEG-DASH MPD XML from the given path.
* On success, print to **stdout** a JSON object mapping each `Representation@id`
  to an **ordered list** of fully-resolved segment URLs.
* On any error, print a clear message to **stderr** via the `log` package,
  disable timestamps with `log.SetFlags(0)`, and exit non-zero.

### URL resolution & BaseURL inheritance
* **Start the resolution chain with the absolute URL
  `http://test.test/test.mpd`.**
* Apply any `BaseURL` elements on top of that in this exact order, using
  `net/url.ResolveReference` each time:
  1. MPD
  2. Period
  3. AdaptationSet
  4. Representation
* If a Representation supplies a `BaseURL` that **doesn’t** end with “/” *and*
  there is no `Segment*` info at any level, output just that single
  fully-resolved URL for the representation.

### Segment generation
* Support **SegmentList** and **SegmentTemplate**.
* **SegmentTemplate rules**
  * Accept `<SegmentTimeline>` *or* the attribute pair `startNumber` /
    `endNumber`.  
    • When only those numbers are present, generate sequentially from
      `startNumber` (default = 1) **through `endNumber`, inclusive**.  
    • If `endNumber` is present *and* a `<SegmentTimeline>` exists, stop once
      `Number` exceeds `endNumber`.
  * Always output the `initialization` URL first (if present), then media
    segments.
* **Token expansion** (with optional `%0Nd` zero-padding) must work in both
  `media` and `initialization` templates and inside `SegmentURL@media` or
  `@sourceURL`:  
  `$RepresentationID$`, `$Number$`, `$Time$`.  
  **NOTE:** `$Time$` must persist across successive `<S>` elements and advance
  by `@d` (duration) unless an explicit `@t` overrides it.

### Period ordering & segment appending
* If the same `Representation@id` appears in multiple **Periods**, append the
  segments in Period order.

### Fallback segment count
* If **both** `SegmentTimeline` and `endNumber` are missing **and** both
  `duration` and `timescale` are present, calculate the number of segments as  
  `ceil(PeriodDurationInSeconds * timescale / duration)`.

### Miscellaneous
* Keep everything in one file (**main.go**) and rely solely on the standard
  library—no third-party packages, no `go:embed`.
* Use clear, idiomatic Go; comment only where it truly clarifies.
* Output nothing except the complete source code.
