package blockmodel

import (
	"os"
	"testing"
)

func TestLoadSimpleModel(t *testing.T) {
	loader := NewLoader("assets-test")
	model, err := loader.LoadModel("block/test_cube")
	if err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	if len(model.Elements) != 1 {
		t.Errorf("Expected 1 element, got %d", len(model.Elements))
	}

	if model.Textures["all"] != "block/stone" {
		t.Errorf("Expected texture 'all' to be 'block/stone', got '%s'", model.Textures["all"])
	}
}

func TestLoadChildModel(t *testing.T) {
	loader := NewLoader("assets-test")
	model, err := loader.LoadModel("block/test_child")
	if err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	if len(model.Elements) != 1 {
		t.Errorf("Expected 1 element from parent, got %d", len(model.Elements))
	}

	if model.Textures["all"] != "block/stone" {
		t.Errorf("Expected texture 'all' to be inherited as 'block/stone', got '%s'", model.Textures["all"])
	}

	if model.Textures["particle"] != "block/dirt" {
		t.Errorf("Expected texture 'particle' to be 'block/dirt', got '%s'", model.Textures["particle"])
	}
}

func TestTextureResolve(t *testing.T) {
	loader := NewLoader("assets-test")
	model, err := loader.LoadModel("block/test_texture_resolve")
	if err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	face := model.Elements[0].Faces["north"]
	if face.Texture != "block/diamond_block" {
		t.Errorf("Expected texture to be resolved to 'block/diamond_block', got '%s'", face.Texture)
	}
}

func TestCache(t *testing.T) {
	loader := NewLoader("assets-test")
	model1, err := loader.LoadModel("block/test_cube")
	if err != nil {
		t.Fatalf("Failed to load model first time: %v", err)
	}

	model2, err := loader.LoadModel("block/test_cube")
	if err != nil {
		t.Fatalf("Failed to load model second time: %v", err)
	}

	if model1 != model2 {
		t.Errorf("Expected the same model instance to be returned from cache")
	}
}

func TestMain(m *testing.M) {
	// Create dummy files for testing
	os.MkdirAll("assets-test/models/block", 0755)

	// test_cube.json
	writeTestFile("assets-test/models/block/test_cube.json", `{
		"textures": { "all": "block/stone" },
		"elements": [ { "from": [0,0,0], "to": [16,16,16], "faces": { "down": { "texture": "#all" } } } ]
	}`)

	// test_child.json
	writeTestFile("assets-test/models/block/test_child.json", `{
		"parent": "block/test_cube",
		"textures": { "particle": "block/dirt" }
	}`)

	// test_texture_resolve.json
	writeTestFile("assets-test/models/block/test_texture_resolve.json", `{
		"textures": { "primary": "block/diamond_block", "secondary": "#primary" },
		"elements": [ { "from": [0,0,0], "to": [16,16,16], "faces": { "north": { "texture": "#secondary" } } } ]
	}`)

	exitCode := m.Run()
	os.RemoveAll("assets-test")
	os.Exit(exitCode)
}

func writeTestFile(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(err)
	}
}
