package agent

import (
	"reflect"
	"testing"
)

func TestParseExternalCollaboratorsDetectsKnownCodingAppsWithoutDuplicates(t *testing.T) {
	processes := `
  101 /Applications/Cursor.app/Contents/MacOS/Cursor
  102 /usr/local/bin/codex --model gpt-5
  103 /usr/local/bin/codex --resume
  104 /opt/homebrew/bin/aider --model test
  105 /Applications/Visual Studio Code.app/Contents/MacOS/Electron
`
	want := []string{"Aider", "Codex", "Cursor", "Visual Studio Code"}
	if got := parseExternalCollaborators(processes); !reflect.DeepEqual(got, want) {
		t.Fatalf("external collaborators = %#v, want %#v", got, want)
	}
}
