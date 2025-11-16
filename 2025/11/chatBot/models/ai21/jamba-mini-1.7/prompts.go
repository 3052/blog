package main

const address = "openrouter.ai/chat?models=ai21/jamba-mini-1.7"

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
      162.7,
      963,
   },
   1: {
      "81:17: declared and not used: representationID",
      162,
      1012,
   },
   2: {
      `10:5: "text/template" imported and not used`,
      161,
      991,
   },
   3: {
      "52:11: undefined: xml",
      161.7,
      1004,
   },
   4: {
      "65:18: undefined: io",
      163.6,
      1034,
   },
   5: {
      "no third party imports",
      162.2,
      1022,
   },
   6: {
      `5:5: "errors" imported and not used`,
      160.3,
      988,
   },
   7: {
      "80:17: declared and not used: representationID",
      161.9,
      995,
   },
   8: {
      "80:59: representation.URLs undefined (type AdaptationSetType has no field or method URLs)",
      160.2,
      1017,
   },
   9: {
      "81:52: representation.URLs undefined (type AdaptationSetType has no field or method URLs)",
      161.3,
      973,
   },
   10: {
      "81:95: mpd.MPD.MPD undefined (type MPDType has no field or method MPD)",
      161.1,
      975,
   },
   11: {
      "81:95: mpd.MPD.MPD undefined (type MPDType has no field or method MPD)",
      161.9,
      1010,
   },
   12: {
      "81:98: cannot use url (variable of struct type URLType) as string value in argument to path.Join",
      161.4,
      1002,
   },
   13: {
      "88:24: undefined: json",
      160.5,
      984,
   },
   14: {
      "103:25: undefined: regexp",
      158.2,
      995,
   },
   15: {
      `8:5: "math" imported and not used`,
      162,
      980,
   },
   16: {
      "61:18: undefined: http",
      160.2,
      998,
   },
   17: {
      "111:14: undefined: mpdURL",
      160.4,
      1095,
   },
   18: {
      "118:14: undefined: mpdURL",
      161.9,
      1105,
   },
   19: {
      "120:14: undefined: mpdURL",
      162.8,
      1112,
   },
   20: {
      "121:14: undefined: mpdURL",
      160.5,
      1115,
   },
   21: {
      "121:14: undefined: mpdURL",
      160.7,
      1131,
   },
}
