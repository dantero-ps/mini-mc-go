package blockmodel

import "encoding/json"

type Model struct {
	Parent           string             `json:"parent"`
	AmbientOcclusion *bool              `json:"ambientocclusion"`
	Textures         map[string]string  `json:"textures"`
	Elements         []Element          `json:"elements"`
	Display          map[string]Display `json:"display"`
	Overrides        []Override         `json:"overrides"`
}

type Element struct {
	From     [3]float32      `json:"from"`
	To       [3]float32      `json:"to"`
	Rotation *Rotation       `json:"rotation"`
	Shade    *bool           `json:"shade"`
	Faces    map[string]Face `json:"faces"`
}

type Rotation struct {
	Origin  [3]float32 `json:"origin"`
	Angle   float32    `json:"angle"`
	Axis    string     `json:"axis"`
	Rescale bool       `json:"rescale"`
}

type Face struct {
	UV        [4]float32 `json:"uv"`
	Texture   string     `json:"texture"`
	CullFace  string     `json:"cullface"`
	Rotation  int        `json:"rotation"`
	TintIndex *int       `json:"tintindex"`
}

type Display struct {
	Rotation    [3]float32 `json:"rotation"`
	Translation [3]float32 `json:"translation"`
	Scale       [3]float32 `json:"scale"`
}

type Override struct {
	Predicate map[string]float32 `json:"predicate"`
	Model     string             `json:"model"`
}

// BlockState defines the blockstate JSON structure. It maps variants of a block to their corresponding models.
type BlockState struct {
	// Variants is a map of variant names to a list of models.
	Variants map[string]BlockStateVariants `json:"variants"`
}

// BlockStateVariants is a custom type to handle the fact that the "variants" field can contain either a single object or an array of objects.
type BlockStateVariants []Variant

func (v *BlockStateVariants) UnmarshalJSON(data []byte) error {
	// First, try to unmarshal as an array
	var variants []Variant
	if err := json.Unmarshal(data, &variants); err == nil {
		*v = variants
		return nil
	}

	// If that fails, try to unmarshal as a single object
	var singleVariant Variant
	if err := json.Unmarshal(data, &singleVariant); err != nil {
		return err
	}

	*v = []Variant{singleVariant}
	return nil
}

type Variant struct {
	Model string `json:"model"`
}
