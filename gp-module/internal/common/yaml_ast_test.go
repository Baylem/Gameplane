package common

import (
	"testing"
)

func TestYAMLNodeLineFinder(t *testing.T) {
	yamlContent := `
apiVersion: gameplane.local/v1alpha1
kind: GameTemplate
spec:
  displayName: Test Game
  image: example/image:latest
  ports:
    - name: game
      containerPort: 25565
      protocol: TCP
    - name: rcon
      containerPort: 25575
      protocol: TCP
`

	root, err := ParseYAMLNode([]byte(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAMLNode failed: %v", err)
	}

	lineImage := FindLineNumber(root, "spec.image")
	if lineImage != 6 {
		t.Errorf("expected line for spec.image to be 6, got %d", lineImage)
	}

	linePort0 := FindLineNumber(root, "spec.ports[0].containerPort")
	if linePort0 != 9 {
		t.Errorf("expected line for spec.ports[0].containerPort to be 9, got %d", linePort0)
	}

	linePort1 := FindLineNumber(root, "spec.ports[1].name")
	if linePort1 != 11 {
		t.Errorf("expected line for spec.ports[1].name to be 11, got %d", linePort1)
	}

	lineUnknown := FindLineNumber(root, "spec.nonexistent")
	if lineUnknown != 0 {
		t.Errorf("expected 0 for nonexistent path, got %d", lineUnknown)
	}
}

func TestYAMLNodeColumnFinder(t *testing.T) {
	yamlContent := `
apiVersion: gameplane.local/v1alpha1
kind: GameTemplate
spec:
  displayName: Test Game
  image: example/image:latest
  ports:
    - name: game
      containerPort: 25565
      protocol: TCP
`

	root, err := ParseYAMLNode([]byte(yamlContent))
	if err != nil {
		t.Fatalf("ParseYAMLNode failed: %v", err)
	}

	// FindColumnNumber returns the VALUE node's column (findMapChild resolves mapping keys to their value nodes).
	// For apiVersion: gameplane.local/v1alpha1, the value starts at column 13.
	colApiVersion := FindColumnNumber(root, "apiVersion")
	if colApiVersion != 13 {
		t.Errorf("expected column 13 for apiVersion, got %d", colApiVersion)
	}

	// For spec.displayName: Test Game, the value starts at column 16 (indented under spec, then the string value).
	colDisplayName := FindColumnNumber(root, "spec.displayName")
	if colDisplayName != 16 {
		t.Errorf("expected column 16 for spec.displayName, got %d", colDisplayName)
	}

	// For spec.ports[0].containerPort: 25565, the value starts at column 22 (deeply indented under sequence and mapping).
	colContainerPort := FindColumnNumber(root, "spec.ports[0].containerPort")
	if colContainerPort != 22 {
		t.Errorf("expected column 22 for spec.ports[0].containerPort, got %d", colContainerPort)
	}

	// Test not-found path: should return 0
	colUnknown := FindColumnNumber(root, "spec.nonexistent")
	if colUnknown != 0 {
		t.Errorf("expected 0 for nonexistent path, got %d", colUnknown)
	}

	// Test that FindColumnNumber matches FindLineNumber structure on success
	lineImage := FindLineNumber(root, "spec.image")
	colImage := FindColumnNumber(root, "spec.image")
	if lineImage == 0 && colImage != 0 {
		t.Errorf("inconsistent: line 0 but column %d", colImage)
	}
	if lineImage > 0 && colImage == 0 {
		t.Errorf("spec.image found on line %d but column is 0", lineImage)
	}
}
