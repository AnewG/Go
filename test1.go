package main

import (
	"fmt"
	"sync"
)

var once11 sync.Once

var once22 sync.Once

var once33 sync.Once

func main() {
	fmt.Println("Hello, playground")
	for i := 0; i < 10; i++ {
		//fmt.Println("？？？")
		//go func() {
			once11.Do(onced)
			once22.Do(onces)
			once33.Do(onces)
			//onced()
			//onces()
			//onces()
			//fmt.Println("213")
		//}()
	}
}

func onces() {
        var a int = 20
        var b int = 10
	fmt.Println(a+b)
}
func onced() {
	fmt.Println("onced")
}
