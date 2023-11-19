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
      homepage: "https://godocs.io/154.pages.dev/api",
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
      homepage: "https://godocs.io/154.pages.dev/encoding",
   },
   {
      description: "Download APK from Google Play or send API requests",
      name: "google",
      topics: []string{
         "android",
         "google-play",
      },
      homepage: "https://godocs.io/154.pages.dev/google",
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
      homepage: "https://godocs.io/154.pages.dev/media",
   },
   {
      description: "Protocol Buffers",
      homepage: "https://godocs.io/154.pages.dev/protobuf",
      name: "protobuf",
   },
   {
      name: "umber",
      homepage: "https://159.pages.dev/umber",
   },
   {
      name: "widevine",
      description: "DRM",
      homepage: "https://godocs.io/154.pages.dev/widevine",
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

