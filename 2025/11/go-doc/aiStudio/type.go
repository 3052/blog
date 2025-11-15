package doc

// Type represents a type definition in a Go package.
type Type struct {
   Name    string
   Doc     string
   Methods []*Function
}
