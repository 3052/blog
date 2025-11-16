package main

const address = "openrouter.ai/chat?models=ai21/jamba-large-1.7"

var prompts = []struct{
   prompt string
   tokens_per_second float64
   token_count int
}{
   0: {
      `return the full Go script that:
      1. Reads an MPEG-DASH MPD XML file path from the CLI: "go run main.go <mpd_file_path>"
      2. Uses "http://test.test/test.mpd" as the initial MPD URL for resolving all relative BaseURLs
      3. Outputs a JSON object mapping each "Representation@id" to a list of fully resolved segment URLs`,
      57,
      1605,
   },
   1: {
      "representation.SegmentList undefined (type Representation has no field or method SegmentList)",
      55.5,
      52,
   },
   2: {
      "representation.SegmentList undefined (type Representation has no field or method SegmentList)",
      55.8,
      787,
   },
   3: {
      "full script",
      57.1,
      167,
   },
   4: {
      "full script",
      58.2,
      142,
   },
   5: {
      "full script",
      43.8,
      435,
   },
   6: {
      "full script",
      52.4,
      1181,
   },
   7: {
      "full script",
      50.6,
      1690,
   },
   8: {
      "Representation.SegmentBase is optional",
      57.7,
      1385,
   },
   9: {
      "Representation.SegmentBase is optional",
      58.9,
      1232,
   },
   10: {
      "full script",
      58.9,
      1958,
   },
   11: {
      "SegmentTemplate can be child of Representation or AdaptationSet",
      51.3,
      2251,
   },
   12: {
      "SegmentTemplate can be child of Representation or AdaptationSet",
      53,
      1922,
   },
   13: {
      "replace $Time$",
      58.9,
      2447,
   },
   14: {
      "145:33: syntax error: unexpected name URL at end of statement",
      51.7,
      720,
   },
   15: {
      "145:33: syntax error: unexpected name URL at end of statement",
      53.6,
      1624,
   },
   16: {
      "145:33: syntax error: unexpected name URL at end of statement",
      57,
      855,
   },
   17: {
      "145:33: syntax error: unexpected name URL at end of statement",
      56.3,
      121,
   },
   18: {
      "145:33: syntax error: unexpected name URL at end of statement",
      58.7,
      1058,
   },
}
