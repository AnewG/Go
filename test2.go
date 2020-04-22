package main

import (
	"fmt"
	"sync"
)

var once sync.Once

var once2 sync.Once

func main() {
	//fmt.Println("Hello, playground")
	for i := 0; i < 10; i++ {
		fmt.Println("？？？")
		go func() {
			once.Do(onced)
			once2.Do(onces)
			fmt.Println("213")
		}()
	}
}

func onces() {
	fmt.Println("onces")
}
func onced() {
	fmt.Println("onced")
}
