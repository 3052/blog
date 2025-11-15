package doc

// FuncDoc holds documentation for a single function.
type FuncDoc struct {
   Name string
   Doc  string
}

// VarDoc holds documentation for a variable or constant.
type VarDoc struct {
   Names []string
   Doc   string
}

// TypeSpecDoc holds documentation for a type specification.
type TypeSpecDoc struct {
   Name string
   Doc  string
}

// TypeDoc holds documentation for a type definition.
type TypeDoc struct {
   Name  string
   Doc   string
   Specs []TypeSpecDoc
}

// PackageDoc holds all the documentation for a package.
type PackageDoc struct {
   Name      string
   Doc       string
   Functions []FuncDoc
   Types     []TypeDoc
   Variables []VarDoc
   Constants []VarDoc
}
