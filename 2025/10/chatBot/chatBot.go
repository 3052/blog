package chatBot

import (
   "encoding/json"
   "net/http"
   "time"
)

func delete_model(m *model) bool {
   if m.ContextLength < 128000 {
      return true
   }
   if m.Endpoint == nil {
      return true
   }
   if m.Endpoint.ModelVariantSlug != m.Slug {
      return true
   }
   const day = 24 * time.Hour
   const month = 30 * day
   // 6 month is 150
   const updated_at = 5 * month
   if time.Since(m.UpdatedAt) >= updated_at {
      return true
   }
   return m.WarningMessage != ""
}

func contains(models []*model, a *model) bool {
   for _, b := range models {
      if b.Slug == a.Slug {
         return true
      }
   }
   return false
}

type model struct {
   ContextLength int       `json:"context_length"`
   Endpoint      *struct { // DELETE
      ModelVariantSlug string `json:"model_variant_slug"`
   } `json:"endpoint"`
   ShortName      string `json:"short_name"`
   Slug           string `json:"slug"`
   UpdatedAt      time.Time `json:"updated_at"`
   WarningMessage string    `json:"warning_message,omitempty"`
}

func find() ([]*model, error) {
   req, _ := http.NewRequest("", "https://openrouter.ai", nil)
   req.URL.Path = "/api/frontend/models/find"
   resp, err := http.DefaultClient.Do(req)
   if err != nil {
      return nil, err
   }
   defer resp.Body.Close()
   var value struct {
      Data struct {
         Models []*model
      }
   }
   err = json.NewDecoder(resp.Body).Decode(&value)
   if err != nil {
      return nil, err
   }
   return value.Data.Models, nil
}
