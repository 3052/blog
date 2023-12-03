package main

import (
   "154.pages.dev/api/justwatch"
   "fmt"
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
      if tag.Language() == f.language {
         v := tag.Variables()
         text, err := v.Text()
         if err != nil {
            return err
         }
         fmt.Println(text)
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
   fmt.Println(text)
   return nil
}
