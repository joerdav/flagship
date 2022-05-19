# flagship - Easily use feature-flags to ship code, with DynamoDB.

[![GoDoc](http://godoc.org/github.com/yosssi/gohtml?status.png)](http://godoc.org/github.com/joerdav/flagship)

## Install

```
go get github.com/joerdav/flagship@latest
```
## Example

``` go
package main

import "github.com/joerdav/flagship"

func main() {
		s, err := flagship.New(context.Background())
		if err != nil {
			t.Errorf("unexpected error got %v", err)
		}
		if s.Bool(context.Background(), "newfeature") {
			log.Println("a new feature")
		} else {
			log.Println("old code")
		}
}
```
