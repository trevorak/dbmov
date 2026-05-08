package output

import (
	"fmt"
	"io"
)

func Fprintf(w io.Writer, format string, args ...any) (n int, err error) {
	return fmt.Fprintf(w, format, args...)
}

func Fprintln(w io.Writer, args ...any) (n int, err error) {
	return fmt.Fprintln(w, args...)
}

func FprintTabbedf(w io.Writer, format string, args ...any) (n int, err error) {
	return Fprintf(w, "         "+format, args...)
}

func Errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
