package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	g := new(game).init()
	defer g.saveAndClose()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		inputLine := scanner.Text()
		command := g.findCommand(strings.Fields(
			strings.Replace(strings.ToLower(inputLine), "ё", "е", -1),
		))
		if command == nil {
			panic(fmt.Errorf(`"%s" command not found`, inputLine))
		}
		g.executeCommand(command)
	}
}
