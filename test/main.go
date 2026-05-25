package main

import (
	"fmt"

	"github.com/brianvoe/gofakeit/v7"
)

func main() {
	for range 1000000 {
		fmt.Println(gofakeit.CompanySuffix())
	}
}
