package main

import (
   "encoding/json"
   "fmt"
   "os"
   "os/exec"
)

type show_entry struct {
   Streams []struct {
      BitRate int `json:"bit_rate,string"`
      CodecType string `json:"codec_type"`
   }
}

func main() {
   os.Chdir("d:/videos")
   entries, err := os.ReadDir(".")
   if err != nil {
      panic(err)
   }
   for i, entry := range entries {
      fmt.Println(len(entries)-i)
      text, err := exec.Command(
         "ffprobe", "-show_entries", "stream", "-of", "json",
         entry.Name(),
      ).Output()
      if err != nil {
         panic(err)
      }
      var show show_entry
      json.Unmarshal(text, &show)
      for _, stream := range show.Streams {
         switch stream.CodecType {
         case "audio":
            if stream.BitRate < 100_000 {
               fmt.Println(entry)
            }
         case "video":
            if stream.BitRate < 2_000_000 {
               fmt.Println(entry)
            }
         }
      }
   }
}
