package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "net/http"
   "strings"
   "time"
)

func format_integer(number int) string {
   numberString := fmt.Sprint(number)
   lengthOfString := len(numberString)
   if lengthOfString <= 3 {
      return numberString
   }
   digits := lengthOfString % 3
   if digits == 0 {
      digits = 3
   }
   var data strings.Builder
   data.WriteString(numberString[:digits])
   for i := digits; i < lengthOfString; i += 3 {
      data.WriteByte(',')
      data.WriteString(numberString[i : i+3])
   }
   return data.String()
}

func (i *InnerTube) do() error {
   play, err := i.Player()
   if err != nil {
      return err
   }
   views := views_per_year(
      play.VideoDetails.ViewCount,
      play.Microformat.PlayerMicroformatRenderer.PublishDate,
   )
   if views >= 10_000_000 {
      fmt.Print(red, " FAIL ", reset)
   } else {
      fmt.Print(green, " PASS ", reset)
   }
   fmt.Print("   ")
   fmt.Print(format_integer(views))
   fmt.Print("   ")
   fmt.Println(play.VideoDetails.VideoId)
   return nil
}

func views_per_year(views int, publish date) int {
   fmt.Println(publish[0])
   years := time.Since(publish[0]).Hours() / 24 / 365
   return int(float64(views) / years)
}

// need `osVersion` this to get the correct:
// This video requires payment to watch
// instead of the invalid:
// This video can only be played on newer versions of Android or other
// supported devices.
type InnerTube struct {
   ContentCheckOk bool `json:"contentCheckOk,omitempty"`
   Context        struct {
      Client struct {
         AndroidSdkVersion int    `json:"androidSdkVersion"`
         ClientName        string `json:"clientName"`
         ClientVersion     string `json:"clientVersion"`
         OsVersion         string `json:"osVersion"`
      } `json:"client"`
   } `json:"context"`
   RacyCheckOk bool   `json:"racyCheckOk,omitempty"`
   VideoId     string `json:"videoId"`
}

const user_agent = "com.google.android.youtube/"

func main() {
   var tube InnerTube
   tube.Context.Client.ClientName = web
   flag.StringVar(&tube.VideoId, "v", "", "video ID")
   flag.Parse()
   if tube.VideoId != "" {
      err := tube.do()
      if err != nil {
         panic(err)
      }
   } else {
      flag.Usage()
   }
}

const (
   web                     = "WEB"
   web_version             = "2.20231219.04.00"
)

const (
   reset = "\x1b[m"
   green = "\x1b[30;102m"
   red   = "\x1b[30;101m"
)

func (i *InnerTube) Player() (*Player, error) {
   i.Context.Client.AndroidSdkVersion = 32
   i.Context.Client.ClientVersion = web_version
   i.Context.Client.OsVersion = "12"
   data, err := json.Marshal(i)
   if err != nil {
      return nil, err
   }
   req, err := http.NewRequest(
      "POST", "https://www.youtube.com/youtubei/v1/player",
      bytes.NewReader(data),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("user-agent", user_agent+i.Context.Client.ClientVersion)
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   play := &Player{}
   err = json.NewDecoder(resp.Body).Decode(play)
   if err != nil {
      return nil, err
   }
   return play, nil
}

type Player struct {
   Microformat struct {
      PlayerMicroformatRenderer struct {
         PublishDate date
      }
   }
   VideoDetails struct {
      VideoId          string
      ViewCount        int `json:",string"`
   }
}

type date [1]time.Time

func (d *date) UnmarshalText(data []byte) error {
   var err error
   d[0], err = time.Parse(time.RFC3339, string(data))
   if err != nil {
      return err
   }
   return nil
}
