package cmd

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/format"
)

// writeOutput converts output to the specified format and writes to stdout or a file.
func writeOutput(output string, outFmt format.Type, outputFile string) error {
	if outFmt == format.PDF {
		if err := format.WritePDF(output, outputFile); err != nil {
			return fmt.Errorf("writing PDF: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[mysh] saved to %s\n", outputFile)
		return nil
	}

	converted, err := format.Convert(output, outFmt)
	if err != nil {
		return err
	}

	if outputFile != "" {
		if err := format.WriteFile(converted, outputFile); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[mysh] saved to %s\n", outputFile)
		return nil
	}

	fmt.Print(converted)
	return nil
}
