package main

import (
   "154.pages.dev/encoding"
   "154.pages.dev/google/play"
   "fmt"
   "os"
   "slices"
   "time"
)

var apps = []application{
   {id: "au.com.stan.and"},
   {id: "com.amcplus.amcfullepisodes"},
   {id: "com.cbs.app"},
   {id: "com.hulu.plus"},
   {id: "com.mubi"},
   {id: "com.nbcuni.nbc"},
   {id: "com.peacocktv.peacockandroid"},
   {id: "com.plexapp.android"},
   {id: "com.roku.web.trc"},
}

func main() {
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   var token play.RefreshToken
   token.Data, err = os.ReadFile(home + "/google-play/token.txt")
   if err != nil {
      panic(err)
   }
   token.Unmarshal()
   var detail play.Details
   if err := detail.Token.Refresh(token); err != nil {
      panic(err)
   }
   detail.Checkin.Data, err = os.ReadFile(home + "/google-play/x86.bin")
   if err != nil {
      panic(err)
   }
   detail.Checkin.Unmarshal()
   for i, app := range apps {
      fmt.Println(app.id)
      detail.Details(app.id, false)
      apps[i].installs, _ = detail.Downloads()
      apps[i].name, _ = detail.Name()
      time.Sleep(99 * time.Millisecond)
   }
   slices.SortFunc(apps, func(a, b application) int {
      return int(b.installs - a.installs)
   })
   for _, app := range apps {
      fmt.Println(app)
   }
}

func (a application) String() string {
   var b []byte
   b = fmt.Append(b, encoding.Cardinal(a.installs))
   b = append(b, ' ')
   b = append(b, a.name...)
   return string(b)
}

type application struct {
   id string
   name string
   installs uint64
}
