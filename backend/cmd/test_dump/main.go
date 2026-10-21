package main

import (
	"fmt"
	"os"
	"strings"

	"my-books/backend/services"
)

func main() {
	path := os.Args[1]
	a, err := services.OpenAZW3(path)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	defer a.Close()
	for i := 0; i < 2; i++ {
		html, err := a.GetContent(i)
		if err != nil {
			fmt.Printf("GetContent(%d) err: %v\n", i, err)
			continue
		}
		s := string(html)
		emptyBr := strings.Count(s, "<div><br")
		emptyBr2 := strings.Count(s, "<div class")
		fmt.Printf("Ch%d len=%d | <div><br 出现=%d | <div class 出现=%d\n", i, len(s), emptyBr, emptyBr2)
		// 打印前 600 字符看结构
		if len(s) > 600 {
			s = s[:600]
		}
		fmt.Println(s)
		fmt.Println()
	}
}
