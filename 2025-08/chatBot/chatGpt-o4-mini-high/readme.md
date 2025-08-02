# chatGpt

https://chatgpt.com

this model fails. after six chances on the first test file, I get this:

> You've hit your limit. Please try again later.

## chance 1

Please provide a complete Go program (main.go) using only the standard library
that:

1. Reads an MPEG-DASH MPD XML file path from the CLI (`go run main.go <mpd_file_path>`)
2. Start the resolution chain with the absolute URL `http://test.test/test.mpd`
3. Appends all segments and outputs a JSON map from each `Representation.ID` to
   its ordered list of fully-resolved segment URLs, with full error handling and
   printing to stdout

## chance 2

BaseURL is string not slice

## chance 3

support **`<SegmentTemplate>`** (at AdaptationSet or Representation) handling

## chance 4

replace `$RepresentationID$`

## chance 5

replace `$Time$`

## chance 6

Chains every `<BaseURL>` at the MPD, Period, and Representation levels via
`net/url.URL.ResolveReference`
