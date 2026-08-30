package main

import (
	"fmt"
	rtdal "github.com/tarekwasfy01/RTDAL---R-Transpiled-Data-Abstraction-Library"
	"log"
)

func main() {
	ctx := rtdal.NewContext()
	value, err := rtdal.Eval(ctx, "x <- c(1, 2, 3); sum(x)")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(value)
}
