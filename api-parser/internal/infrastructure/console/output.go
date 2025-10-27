package console

import (
	"fmt"
	"io"
)

type ConsoleOutput struct {
	writer io.Writer
}

func NewConsoleOutput(w io.Writer) ConsoleOutput {
	return ConsoleOutput{writer: w}
}

func (c ConsoleOutput) Print(v any) error {
	_, err := fmt.Fprintln(c.writer, v)
	return err
}
