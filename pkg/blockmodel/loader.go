package blockmodel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Loader struct {
	assetsPath string
	modelCache map[string]*Model
}

func NewLoader(assetsPath string) *Loader {
	return &Loader{
		assetsPath: assetsPath,
		modelCache: make(map[string]*Model),
	}
}

func (l *Loader) LoadModel(name string) (*Model, error) {
	if !strings.Contains(name, "/") {
		name = "block/" + name
	}

	if model, ok := l.modelCache[name]; ok {
		return model, nil
	}

	path := filepath.Join(l.assetsPath, "models", name+".json")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read model file: %w", err)
	}

	var model Model
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("could not unmarshal model json: %w", err)
	}

	if model.Parent != "" {
		parentName := model.Parent
		if strings.HasPrefix(parentName, "builtin/") {
			l.modelCache[name] = &model
			return &model, nil
		}

		parent, err := l.LoadModel(parentName)
		if err != nil {
			return nil, fmt.Errorf("could not load parent model '%s': %w", parentName, err)
		}

		if model.AmbientOcclusion == nil {
			model.AmbientOcclusion = parent.AmbientOcclusion
		}
		if len(model.Elements) == 0 {
			model.Elements = parent.Elements
		}
		for key, val := range parent.Textures {
			if _, ok := model.Textures[key]; !ok {
				model.Textures[key] = val
			}
		}
	}

	l.resolveTextures(&model)
	l.modelCache[name] = &model
	return &model, nil
}

func (l *Loader) resolveTextures(m *Model) {
	for i := range m.Elements {
		for faceName, face := range m.Elements[i].Faces {
			originalTexture := face.Texture
			resolvedTexture := l.ResolveTexture(originalTexture, m)
			if resolvedTexture != originalTexture {
				face.Texture = resolvedTexture
				m.Elements[i].Faces[faceName] = face
			}
		}
	}
}

func (l *Loader) ResolveTexture(textureName string, m *Model) string {
	for i := 0; i < 10 && strings.HasPrefix(textureName, "#"); i++ {
		key := strings.TrimPrefix(textureName, "#")
		if resolved, ok := m.Textures[key]; ok {
			textureName = resolved
		} else {
			break
		}
	}
	return textureName
}

func (l *Loader) LoadBlockState(name string) (*BlockState, error) {
	path := filepath.Join(l.assetsPath, "blockstates", name+".json")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read blockstate file: %w", err)
	}

	var blockState BlockState
	if err := json.Unmarshal(data, &blockState); err != nil {
		return nil, fmt.Errorf("could not unmarshal blockstate json: %w", err)
	}

	return &blockState, nil
}
