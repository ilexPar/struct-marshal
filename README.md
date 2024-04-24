# struct-marshal
Utility to translate between two go structs using json encoding

Intended to translate between API objects and the internal system definitions using "jsonpath" tag to map fields.

A real world example would be to translate between Kubernetes API objects and a custom struct internal to your application.

Example translating an internal object to a deployment:
```go
import (
    sm "github.com/ilexPar/struct-marshal/pkg"
    apps "k8s.io/api/apps/v1"
)

type InternalObject struct {
    Name   string `sm:"metadata.name"`
    Image  string `sm:"spec.template.spec.containers[0].image"`
    Memory string `sm:"spec.template.spec.containers[0].resources.limits.memory"`
}

func main() {
    src := &apps.Deployment{}
    dst := &InternalObject{}
    sm.StructUnmarshal(src, dst) // now dst should have been populated with the expected values from src
}
```

Example translating a deployment to an internal object:

```go
import (
    sm "github.com/ilexPar/struct-marshal/pkg"
    apps "k8s.io/api/apps/v1"
)

type InternalObject struct {
    Name   string `sm:"metadata.name"`
    Image  string `sm:"spec.template.spec.containers[0].image"`
    Memory string `sm:"spec.template.spec.containers[0].resources.limits.memory"`
}

func main() {
    dst := &apps.Deployment{}
    src := &InternalObject{
        Name: "my-dpl",
        Image: "nginx",
        Memory: "128Mi",
    }
    sm.StructMmarshal(src, dst) // now dst should have been populated with the expected values from src
}
```