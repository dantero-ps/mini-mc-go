package blockmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSharedParentMutation(t *testing.T) {
	// Setup test files:
	// parent.json: abstract parent with #texture
	// child1.json: defines texture="skin1"
	// child2.json: defines texture="skin2"
	//
	// If shallow copy bug exists:
	// Loading child1 resolves parent element to skin1.
	// Loading child2 executes resolveTextures. If it got modified parent elements, it sees skin1.
	// But resolveTextures re-resolves #texture.
	// The problem is if the element ALREADY has skin1 (not #texture), then resolveTextures sees skin1 and stops.

	dir := "assets-test-sharing/models/block"
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll("assets-test-sharing")

	writeFile(filepath.Join(dir, "parent.json"), `{
		"textures": { "dummy": "ignore" },
		"elements": [ { "from": [0,0,0], "to": [16,16,16], "faces": { "up": { "texture": "#all" } } } ]
	}`)

	writeFile(filepath.Join(dir, "child1.json"), `{
		"parent": "block/parent",
		"textures": { "all": "block/skin1" }
	}`)

	writeFile(filepath.Join(dir, "child2.json"), `{
		"parent": "block/parent",
		"textures": { "all": "block/skin2" }
	}`)

	loader := NewLoader("assets-test-sharing")

	// Load Child 1
	c1, err := loader.LoadModel("block/child1")
	if err != nil {
		t.Fatalf("Failed to load child1: %v", err)
	}
	// Verify c1 resolved to skin1
	if c1.Elements[0].Faces["up"].Texture != "block/skin1" {
		t.Errorf("Child1 should have skin1, got %s", c1.Elements[0].Faces["up"].Texture)
	}

	// Load Child 2
	c2, err := loader.LoadModel("block/child2")
	if err != nil {
		t.Fatalf("Failed to load child2: %v", err)
	}
	// Verify c2 resolved to skin2
	// If parent was mutated, c2 might see skin1 or keep skin1 if resolve logic short-circuits
	if c2.Elements[0].Faces["up"].Texture != "block/skin2" {
		t.Errorf("Child2 should have skin2, got %s. Likely parent pollution.", c2.Elements[0].Faces["up"].Texture)
	}

	// Double check parent model cache integrity (optional, depends on implementation but good to know)
	parent, _ := loader.LoadModel("block/parent")
	if parent.Elements[0].Faces["up"].Texture != "#all" {
		t.Errorf("Parent model in cache was mutated! Got %s", parent.Elements[0].Faces["up"].Texture)
	}
}

func writeFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(err)
	}
}
