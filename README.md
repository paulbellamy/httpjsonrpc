# httpjsonrpc

A golang `net/rpc.Codec` which makes HTTP requests with json-rpc
bodies.

For example, given an API with requests like:

```http
POST /endpoint HTTP/1.1
Content-Type: application/json
Content-Length: 63

{"jsonrpc": "2.0", "id": 1, "method": "add", "params": [1, 2]}
```

and responses like:

```http
HTTP/1.1 200 OK
Date: Mon, 08 Jan 2018 10:08:54 GMT
Content-Type: application/json
Content-Length: 36

{"jsonrpc":"2.0","id":1,"result":3}
```

you can do:

```Go
import (
  "fmt"
  "net/rpc"

  "github.com/paulbellamy/httpjsonrpc"
)

func main() {
  client := rpc.NewClientWithCodec(&httpjsonrpc.Codec{
    URL: "http://example.com/endpoint",
  })

  var result int
  if err := client.Call("add", []int{1, 2}, &result); err != nil {
    log.Fatal(err)
  }
  fmt.Println("Result:", result)
}
```
