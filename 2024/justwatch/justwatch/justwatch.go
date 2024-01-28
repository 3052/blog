package main

import (
   "blog/justwatch"
   "log/slog"
   "net/url"
   "time"
)

func (f flags) stream() error {
   address, err := url.Parse(f.address)
   if err != nil {
      return err
   }
   content, err := justwatch.New_URLs(address.Path)
   if err != nil {
      return err
   }
   offer := make(justwatch.Offers)
   for _, tag := range content.Href_Lang_Tags {
      slog.Debug(tag.Href_Lang)
      if !justwatch.Blacklist[tag.Href_Lang] {
         v := tag.Variables()
         text, err := v.Text()
         if err != nil {
            return err
         }
         slog.Info(text)
         detail, err := v.Details()
         if err != nil {
            return err
         }
         offer.Add(v.Country_Code, detail)
         time.Sleep(f.sleep)
      }
   }
   text, err := offer.Stream().Text()
   if err != nil {
      return err
   }
   slog.Info(text)
   return nil
}
