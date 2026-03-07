package mapdata

import (
	"strings"
	"testing"
)

func TestParseDotPositionsHandlesLongLine(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 80_000)
	input := []byte("123 [label=\"" + long + "\" pos=\"10.7,20.9\"];\n")

	positions, err := parseDotPositions(input)
	if err != nil {
		t.Fatalf("parseDotPositions() error = %v", err)
	}
	pos, ok := positions[123]
	if !ok {
		t.Fatalf("parseDotPositions() missing node 123")
	}
	if pos.x != 10 || pos.y != 20 {
		t.Fatalf("parseDotPositions() = %+v, want {x:10 y:20}", pos)
	}
}

func TestParsePlainPositionsHandlesLongLine(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", 80_000)
	input := []byte("node 456 15.9 25.2 0 0 " + long + "\n")

	positions, err := parsePlainPositions(input)
	if err != nil {
		t.Fatalf("parsePlainPositions() error = %v", err)
	}
	pos, ok := positions[456]
	if !ok {
		t.Fatalf("parsePlainPositions() missing node 456")
	}
	if pos.x != 15 || pos.y != 25 {
		t.Fatalf("parsePlainPositions() = %+v, want {x:15 y:25}", pos)
	}
}
