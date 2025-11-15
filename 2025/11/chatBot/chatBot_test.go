package chatBot

import (
   "encoding/json"
   "fmt"
   "log"
   "os"
   "slices"
   "strings"
   "testing"
)

func TestSlug(t *testing.T) {
   data, err := os.ReadFile(name)
   if err != nil {
      t.Fatal(err)
   }
   var models []*model
   err = json.Unmarshal(data, &models)
   if err != nil {
      t.Fatal(err)
   }
   models = slices.DeleteFunc(models, delete_model)
   slices.SortFunc(models, func(a, b *model) int {
      return strings.Compare(a.Slug, b.Slug)
   })
   for _, modelVar := range models {
      fmt.Println(modelVar.Slug)
   }
}

func TestContains(t *testing.T) {
   log.SetFlags(log.Ltime)
   // A
   data, err := os.ReadFile(name)
   if err != nil {
      t.Fatal(err)
   }
   var modelsA []*model
   err = json.Unmarshal(data, &modelsA)
   if err != nil {
      t.Fatal(err)
   }
   modelsA = slices.DeleteFunc(modelsA, delete_model)
   // B
   modelsB, err := find()
   if err != nil {
      t.Fatal(err)
   }
   modelsB = slices.DeleteFunc(modelsB, delete_model)
   for _, modelA := range modelsA {
      if !contains(modelsB, modelA) {
         log.Println("removed", modelA.Slug)
      }
   }
   for _, modelB := range modelsB {
      if !contains(modelsA, modelB) {
         log.Println("added", modelB.Slug)
      }
   }
}

func TestWrite(t *testing.T) {
   models, err := find()
   if err != nil {
      t.Fatal(err)
   }
   data, err := json.MarshalIndent(models, "", " ")
   if err != nil {
      t.Fatal(err)
   }
   err = os.WriteFile(name, data, os.ModePerm)
   if err != nil {
      t.Fatal(err)
   }
}

const name = "chatBot.json"
