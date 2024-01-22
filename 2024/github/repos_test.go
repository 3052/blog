package github

import (
   "fmt"
   "os"
   "testing"
   "time"
)

var repos = []repository{
   {
      name: "api",
      topics: []string{
         "github",
         "justwatch",
         "mullvad",
         "musicbrainz",
      },
   },
   {
      name: "encoding",
      description: "Data parsers and formatters",
      topics: []string{
         "dash",
         "hls",
         "json",
         "mp4",
         "xml",
      },
   },
   {
      description: "Download APK from Google Play or send API requests",
      name: "google",
      topics: []string{
         "android",
         "google-play",
      },
   },
   {
      name: "media",
      description: "Download media or send API requests",
      topics: []string{
         "amc",
         "bandcamp",
         "cbc-gem",
         "nbc",
         "paramount",
         "roku",
         "soundcloud",
         "youtube",
      },
   },
   {
      description: "Protocol Buffers",
      name: "protobuf",
   },
   {
      name: "widevine",
      description: "DRM",
   },
}

const sleep = 99*time.Millisecond

func Test_Repo(t *testing.T) {
   home, err := os.UserHomeDir()
   if err != nil {
      t.Fatal(err)
   }
   u, err := user_info(home + "/github.json")
   if err != nil {
      t.Fatal(err)
   }
   for _, repo := range repos {
      fmt.Println(repo.name)
      err := repo.set_actions(u)
      if err != nil {
         t.Fatal(err)
      }
      time.Sleep(sleep)
      if err := repo.set_description(u); err != nil {
         t.Fatal(err)
      }
      time.Sleep(sleep)
      if repo.topics != nil {
         err := repo.set_topics(u)
         if err != nil {
            t.Fatal(err)
         }
         time.Sleep(sleep)
      }
   }
}

