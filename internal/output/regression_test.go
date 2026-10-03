package output

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/tasnimzotder/portman/internal/model"
)

func TestDelimitedEscapesProcessNames(t *testing.T) {
	name := "app,\"quoted\"\twith\nnewline"
	for _, format := range []string{"csv", "tsv"} {
		text := FormatDelimited([]model.Listener{{Port: 34567, Process: &model.Process{Name: name}}}, format, false)
		reader := csv.NewReader(strings.NewReader(text))
		if format == "tsv" {
			reader.Comma = '\t'
		}
		records, err := reader.ReadAll()
		if err != nil || len(records) != 2 || records[1][4] != name {
			t.Fatalf("%s records=%v err=%v", format, records, err)
		}
	}
}
