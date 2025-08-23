package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func info(format string, args ...interface{}) {
	fmt.Printf("%s\t", color.GreenString("INFO:"))
	fmt.Printf(format+"\n", args...)
}

func warn(format string, args ...interface{}) {
	fmt.Printf("%s\t", color.YellowString("WARN:"))
	fmt.Printf(format+"\n", args...)
}

func ask(format string, args ...interface{}) {
	fmt.Printf("%s\t", color.BlueString("ASK:"))
	fmt.Printf(format+": ", args...)
}

func hint(format string, args ...interface{}) {
	fmt.Printf("%s\t", color.CyanString("HINT:"))
	fmt.Printf(format+"\n", args...)
}

func errorr(format string, args ...interface{}) {
	fmt.Printf("%s\t", color.RedString("ERROR:"))
	fmt.Printf(format+"\n", args...)
	os.Exit(1)
}

func readAnswer() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer, nil
}
