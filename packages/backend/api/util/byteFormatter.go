package util

import (
	"fmt"
	"strings"
)

const OneKb = 1024.0
const OneMb = 1024 * OneKb
const OneGb = 1024 * OneMb

type UnitSizeTuple struct {
	name string
	size float64
}

var UnitSizeTuples = []UnitSizeTuple{
	{name: "GB", size: OneGb},
	{name: "MB", size: OneMb},
	{name: "KB", size: OneKb},
}

func FormatBytes(bytes int64) string {
	size := float64(bytes)
	for _, tuple := range UnitSizeTuples {
		if tuple.size < size {
			var quotient = size / tuple.size
			if quotient > 0 {
				formatted := fmt.Sprintf("%.4f", quotient)
				dotPos := strings.Index(formatted, ".")

				if dotPos != -1 {
					done := false
					for done == false {
						if last := len(formatted) - 1; last >= dotPos && formatted[last] == '0' {
							formatted = formatted[:last]
						} else {
							done = true
						}
					}
				}

				return fmt.Sprintf("%s %s", formatted, tuple.name)
			}
		}
	}

	return fmt.Sprintf("%.2f bytes", size)
}
