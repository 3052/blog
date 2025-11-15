package samplepkg

// Pi is a mathematical constant.
const Pi = 3.14159

// MaxConnections is the maximum number of allowed connections.
const MaxConnections = 100

var (
   // DefaultPort is the default port the server listens on.
   DefaultPort = 8080
)

// Add integers safely.
// This is a package-level function.
func Add(a, b int) int {
   return a + b
}
