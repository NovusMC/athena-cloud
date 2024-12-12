package common

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/printer"
	"io"
)

func format(attr color.Attribute) string {
	return fmt.Sprintf("\u001B[%dm", attr)
}

func EncodeYamlColorized(obj any, out io.Writer) error {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}
	tokens := lexer.Tokenize(string(bytes))
	var p printer.Printer
	p.Bool = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiWhite),
			Suffix: format(color.Reset),
		}
	}
	p.Number = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiWhite),
			Suffix: format(color.Reset),
		}
	}
	p.MapKey = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgBlue),
			Suffix: format(color.Reset),
		}
	}
	p.Anchor = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.Alias = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.String = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiWhite),
			Suffix: format(color.Reset),
		}
	}
	p.Comment = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiBlack),
			Suffix: format(color.Reset),
		}
	}
	_, err = out.Write([]byte(p.PrintTokens(tokens) + "\n"))
	if err != nil {
		return fmt.Errorf("failed to write colorized yaml: %w", err)
	}
	return nil
}
