package github

import (
   "bytes"
   "encoding/json"
   "errors"
   "net/http"
   "strings"
)

func (r repository) set_actions(m map[string]string) error {
   body, err := json.Marshal(map[string]bool{
      "enabled": false,
   })
   if err != nil {
      return err
   }
   req, err := http.NewRequest(
      "PUT", "https://api.github.com", bytes.NewReader(body),
   )
   if err != nil {
      return err
   }
   req.SetBasicAuth(m["username"], m["password"])
   req.URL.Path = func() string {
      var b strings.Builder
      b.WriteString("/repos/")
      b.WriteString(m["username"])
      b.WriteByte('/')
      b.WriteString(r.name)
      b.WriteString("/actions/permissions")
      return b.String()
   }()
   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusNoContent {
      return errors.New(res.Status)
   }
   return nil
}

func (r repository) set_description(m map[string]string) error {
   body, err := json.Marshal(map[string]any{
      "description": r.description,
      "has_projects": false,
      "has_wiki": false,
      "homepage": r.homepage,
   })
   if err != nil {
      return err
   }
   req, err := http.NewRequest(
      "PATCH", "https://api.github.com", bytes.NewReader(body),
   )
   if err != nil {
      return err
   }
   req.SetBasicAuth(m["username"], m["password"])
   req.URL.Path = "/repos/" + m["username"] + "/" + r.name
   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return errors.New(res.Status)
   }
   return nil
}

func (r repository) set_topics(m map[string]string) error {
   body, err := json.Marshal(map[string][]string{
      "names": r.topics,
   })
   if err != nil {
      return err
   }
   req, err := http.NewRequest(
      "PUT", "https://api.github.com", bytes.NewReader(body),
   )
   if err != nil {
      return err
   }
   req.SetBasicAuth(m["username"], m["password"])
   req.URL.Path = "/repos/" + m["username"] + "/" + r.name + "/topics"
   res, err := http.DefaultClient.Do(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return errors.New(res.Status)
   }
   return nil
}

type repository struct {
   description string
   homepage string
   name string
   topics []string
}
