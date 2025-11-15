package doc

// Package represents a Go package.
type Package struct {
   Name      string
   Doc       string
   Functions []*Function
   Types     []*Type
}
