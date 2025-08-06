# qwen

- https://chat.qwen.ai
- https://github.com/QwenLM/Qwen3/issues/1592
- https://wikipedia.org/wiki/Qwen

## prompt 1

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs
