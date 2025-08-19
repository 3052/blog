# gpt-oss

the web client gives no indication of when it is finished with a response, so
you have to constantly scroll until you see it finish. you can set scrolling with
middle click, but even with that your mouse is captured so you literally cannot
use your mouse until the response is done

## prompt 1

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs
