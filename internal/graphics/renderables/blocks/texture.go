package blocks

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
	"os"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// TextureAtlas manages the texture array for blocks
type TextureAtlas struct {
	TextureID uint32
}

var GlobalTextureAtlas *TextureAtlas

// InitTextureAtlas loads all block textures into a GL_TEXTURE_2D_ARRAY
func InitTextureAtlas() error {
	// Initialize registry first to populate TextureNames
	registry.InitRegistry()

	// List of textures to load from registry
	textureFiles := registry.TextureNames
	if len(textureFiles) == 0 {
		return fmt.Errorf("no textures found in registry")
	}

	// Load images
	var images []*image.RGBA
	width, height := 0, 0

	for _, name := range textureFiles {
		path := "assets/textures/blocks/" + name
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open texture %s: %v", path, err)
		}

		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to decode texture %s: %v", path, err)
		}

		rgba := image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

		// Check bounds and crop if necessary (e.g. animated textures like water_flow)
		// We expect square textures for the atlas (e.g. 16x16, 32x32).
		// If height > width, we take the top square.

		finalImg := rgba
		dx := rgba.Bounds().Dx()
		dy := rgba.Bounds().Dy()

		if dy > dx {
			// Crop top square
			rect := image.Rect(0, 0, dx, dx)
			cropped := image.NewRGBA(rect)
			draw.Draw(cropped, rect, rgba, image.Point{0, 0}, draw.Src)
			finalImg = cropped
			// Update dimensions check to use cropped size
			dy = dx
		}

		if width == 0 {
			width = dx
			height = dy
		} else if dx != width || dy != height {
			// Resize/Resample if mismatch (Nearest Neighbor)
			// e.g. 32x32 -> 16x16
			log.Printf("Resizing texture %s from %dx%d to %dx%d", name, dx, dy, width, height)

			resized := image.NewRGBA(image.Rect(0, 0, width, height))

			// Simple Nearest Neighbor scaling
			xRatio := float32(dx) / float32(width)
			yRatio := float32(dy) / float32(height)

			for y := 0; y < height; y++ {
				for x := 0; x < width; x++ {
					srcX := int(float32(x) * xRatio)
					srcY := int(float32(y) * yRatio)

					// Clamp
					if srcX >= dx {
						srcX = dx - 1
					}
					if srcY >= dy {
						srcY = dy - 1
					}

					resized.Set(x, y, finalImg.At(srcX, srcY))
				}
			}
			finalImg = resized
		}

		images = append(images, finalImg)
	}

	// Create Texture Array
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, texture)

	// Storage
	gl.TexImage3D(
		gl.TEXTURE_2D_ARRAY,
		0,
		gl.RGBA8,
		int32(width),
		int32(height),
		int32(len(images)),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil,
	)

	// Upload layers
	for i, img := range images {
		gl.TexSubImage3D(
			gl.TEXTURE_2D_ARRAY,
			0,
			0, 0, int32(i),
			int32(width),
			int32(height),
			1,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(img.Pix),
		)
	}

	// Parameters
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.NEAREST_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_T, gl.REPEAT)

	gl.GenerateMipmap(gl.TEXTURE_2D_ARRAY)

	// Anisotropic filtering if available
	var maxAnisotropy float32
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &maxAnisotropy)
	if maxAnisotropy > 0 {
		gl.TexParameterf(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAX_ANISOTROPY, maxAnisotropy)
	}

	GlobalTextureAtlas = &TextureAtlas{
		TextureID: texture,
	}

	log.Printf("Loaded %d textures into array (size: %dx%d)", len(images), width, height)
	return nil
}

// GetTextureLayer returns the layer index for a block face
func GetTextureLayer(blockType world.BlockType, face world.BlockFace) int {
	return registry.GetTextureLayer(blockType, face)
}
