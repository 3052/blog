# qwen

https://wikipedia.org/wiki/Qwen

free:

https://openrouter.ai/qwen/qwen3-coder:free

no idea on cost:

https://huggingface.co/Qwen/Qwen3-Coder-30B-A3B-Instruct

## prompt 1

return the full Go script that:
- Reads an MPEG-DASH MPD XML file path from the CLI: `go run main.go <mpd_file_path>`
- Uses `http://test.test/test.mpd` as the initial MPD URL for resolving all relative BaseURLs
- Outputs a JSON object mapping each `Representation@id` to a list of fully resolved segment URLs

## prompt 2

standard library only
