package main

import (
   "flag"
   "fmt"
   "strconv"
   "time"
)

func main() {
   var tube InnerTube
   tube.Context.Client.ClientName = android
   flag.StringVar(&tube.VideoId, "v", "", "video ID")
   flag.Parse()
   if tube.VideoId != "" {
      play, err := tube.Player()
      if err != nil {
         panic(err)
      }
      views, err := views_per_year(play)
      if err != nil {
         panic(err)
      }
      fmt.Println(views)
   } else {
      flag.Usage()
   }
}

func views_per_year(play *Player) (string, error) {
   views := func() float64 {
      t := play.Microformat.PlayerMicroformatRenderer.PublishDate.Time
      y := time.Since(t).Hours() / 24 / 365
      return float64(play.VideoDetails.ViewCount) / y
   }()
   var data []byte
   if views >= 10_000_000 {
      data = append(data, red...)
      data = append(data, " FAIL "...)
   } else {
      data = append(data, green...)
      data = append(data, " PASS "...)
   }
   data = append(data, reset...)
   data = append(data, "   "...)
   data = strconv.AppendFloat(data, views, 'f', 0, 64)
   data = append(data, "   "...)
   data = append(data, play.VideoDetails.VideoId...)
   return string(data), nil
}

const (
   reset = "\x1b[m"
   green = "\x1b[30;102m"
   red = "\x1b[30;101m"
)
