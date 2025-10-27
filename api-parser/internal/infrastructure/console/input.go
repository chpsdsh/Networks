package console

import (
	"bufio"
	"io"
	"strings"
)

type ConsoleInput struct {
	io.Reader
}

func NewConsoleInput(r io.Reader) ConsoleInput {
	return ConsoleInput{Reader: r}
}

func (c ConsoleInput) InputData() (string, error) {
	scanner := bufio.NewScanner(c)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", scanner.Err()
}
