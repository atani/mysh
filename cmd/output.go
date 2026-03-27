package cmd

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/db"
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

// writeOutputStructured converts structured data (headers/rows) directly to the
// specified format without an intermediate tabular string serialization.
func writeOutputStructured(headers []string, rows [][]string, outFmt format.Type, outputFile string) error {
	if outFmt == format.PDF {
		if err := format.WritePDFResult(headers, rows, outputFile); err != nil {
			return fmt.Errorf("writing PDF: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[mysh] saved to %s\n", outputFile)
		return nil
	}

	if outFmt == format.Plain {
		output := db.FormatTabular(headers, rows)
		if outputFile != "" {
			if err := format.WriteFile(output, outputFile); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "[mysh] saved to %s\n", outputFile)
			return nil
		}
		fmt.Print(output)
		return nil
	}

	converted, err := format.ConvertResult(headers, rows, outFmt)
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
