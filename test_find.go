// +build ignore

package main

import (
	"encoding/json"
	"fmt"

	"github.com/navicore/mcpterm-go/pkg/tools"
)

func main() {
	executor := tools.NewToolExecutor()

	// Test 1: Basic find with type and name
	input1 := tools.FindInput{
		Directory: ".",
		Type:      "f",
		Name:      "*.go",
		Maxdepth:  2,
	}

	jsonInput1, _ := json.Marshal(input1)
	result1, err := executor.ExecuteTool("find", jsonInput1)
	if err != nil {
		fmt.Printf("Test 1 error: %v\n", err)
	} else {
		fmt.Println("Test 1 (Basic find):")
		files1 := result1.([]string)
		for i, file := range files1 {
			if i >= 5 {
				fmt.Println("...")
				break
			}
			fmt.Println(file)
		}
		fmt.Printf("Total: %d files found\n\n", len(files1))
	}

	// Test 2: Find with mtime (files modified in the last day)
	input2 := tools.FindInput{
		Directory: ".",
		Type:      "f",
		Name:      "*.go",
		Mtime:     "-1",
		Maxdepth:  2,
	}

	jsonInput2, _ := json.Marshal(input2)
	result2, err := executor.ExecuteTool("find", jsonInput2)
	if err != nil {
		fmt.Printf("Test 2 error: %v\n", err)
	} else {
		fmt.Println("Test 2 (With mtime):")
		files2 := result2.([]string)
		for i, file := range files2 {
			if i >= 5 {
				fmt.Println("...")
				break
			}
			fmt.Println(file)
		}
		fmt.Printf("Total: %d files found\n\n", len(files2))
	}

	// Test 3: Find with size (files larger than 1KB)
	input3 := tools.FindInput{
		Directory: ".",
		Type:      "f",
		Name:      "*.go",
		Size:      "+1k",
		Maxdepth:  2,
	}

	jsonInput3, _ := json.Marshal(input3)
	result3, err := executor.ExecuteTool("find", jsonInput3)
	if err != nil {
		fmt.Printf("Test 3 error: %v\n", err)
	} else {
		fmt.Println("Test 3 (With size):")
		files3 := result3.([]string)
		for i, file := range files3 {
			if i >= 5 {
				fmt.Println("...")
				break
			}
			fmt.Println(file)
		}
		fmt.Printf("Total: %d files found\n\n", len(files3))
	}

	// Test 4: Find with path exclusion
	input4 := tools.FindInput{
		Directory: ".",
		Type:      "f",
		Path:      "!*/\\.git/*",
		Maxdepth:  3,
	}

	// Test 5: Complex BSD find with multiple options
	input5 := tools.FindInput{
		Directory: ".",
		Type:      "f",
		Name:      "*.go",
		Size:      "+1k",
		Mtime:     "-7",
		Maxdepth:  3,
	}

	jsonInput4, _ := json.Marshal(input4)
	result4, err := executor.ExecuteTool("find", jsonInput4)
	if err != nil {
		fmt.Printf("Test 4 error: %v\n", err)
	} else {
		fmt.Println("Test 4 (With path exclusion):")
		files4 := result4.([]string)
		for i, file := range files4 {
			if i >= 5 {
				fmt.Println("...")
				break
			}
			fmt.Println(file)
		}
		fmt.Printf("Total: %d files found\n\n", len(files4))
	}

	// Run test 5
	jsonInput5, _ := json.Marshal(input5)
	result5, err := executor.ExecuteTool("find", jsonInput5)
	if err != nil {
		fmt.Printf("Test 5 error: %v\n", err)
	} else {
		fmt.Println("Test 5 (Complex BSD find with multiple options):")
		files5 := result5.([]string)
		for i, file := range files5 {
			if i >= 5 {
				fmt.Println("...")
				break
			}
			fmt.Println(file)
		}
		fmt.Printf("Total: %d files found\n\n", len(files5))
	}
}