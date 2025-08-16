package graphics

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// FontCharacter describes a single character's placement and metrics within the atlas
type FontCharacter struct {
	// Pixel coordinates of the glyph in the atlas texture (top-left origin)
	AtlasX float32
	AtlasY float32
	// Glyph bitmap size in pixels
	Width  float32
	Height float32
	// Bearing (offset from baseline) in pixels
	BearingX float32
	BearingY float32
	// Advance in pixels (already converted from 26.6 if sourced that way)
	Advance int
}

// FontAtlasInfo contains the OpenGL texture and per-glyph metadata
type FontAtlasInfo struct {
	TextureID  uint32
	AtlasW     int
	AtlasH     int
	Characters map[rune]FontCharacter
}

// BuildFontAtlas loads a TrueType font file and bakes an ASCII glyph set into an OpenGL texture atlas.
// fontPixels is the target pixel size for glyphs.
func BuildFontAtlas(fontPath string, fontPixels int) (*FontAtlasInfo, error) {
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("read font: %w", err)
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: float64(fontPixels), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("new face: %w", err)
	}
	defer func() { _ = face.Close() }()

	// Character set: ASCII printable range 32..126
	var runes []rune
	for r := rune(32); r <= rune(1000); r++ {
		runes = append(runes, r)
	}

	// First pass: measure to estimate atlas size
	// Simple row packer with fixed width; choose width ~ 1024, grow height as needed
	atlasW := 1090
	padding := 1
	// Estimate maximum glyph height to choose initial row height
	maxH := 0
	for _, r := range runes {
		_, mask, _, _, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok || mask == nil {
			continue
		}
		dims := mask.Bounds().Size()
		if dims.Y > maxH {
			maxH = dims.Y
		}
	}
	if maxH == 0 {
		maxH = fontPixels
	}

	// Compute required height by packing in rows
	rowH := maxH + padding
	offsetX := 0
	requiredH := rowH
	for _, r := range runes {
		_, mask, _, _, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok || mask == nil {
			continue
		}
		dims := mask.Bounds().Size()
		w := dims.X
		if offsetX+w+padding > atlasW {
			requiredH += rowH
			offsetX = 0
		}
		offsetX += w + padding
	}
	// Round height up to next power-of-two (nice for older GPUs)
	atlasH := 1090

	// Create a single-channel alpha image as the atlas canvas
	atlasImg := image.NewAlpha(image.Rect(0, 0, atlasW, atlasH))

	characters := make(map[rune]FontCharacter)

	// Second pass: render each glyph into the atlas and record metrics
	offsetX, offsetY, rowHeight := 0, 0, 0
	for _, r := range runes {
		dr, mask, maskp, advance, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok || mask == nil {
			continue
		}
		gw := dr.Dx()
		gh := dr.Dy()
		if gw == 0 || gh == 0 {
			// Space or non-drawable glyph; still record advance
			characters[r] = FontCharacter{
				AtlasX:   float32(offsetX),
				AtlasY:   float32(offsetY),
				Width:    0,
				Height:   0,
				BearingX: float32(dr.Min.X),
				BearingY: float32(-dr.Min.Y),
				Advance:  int(math.Round(float64(advance) / 64.0)),
			}
			continue
		}

		if offsetX+gw > atlasW {
			offsetX = 0
			offsetY += rowHeight + padding
			rowHeight = 0
		}

		dstRect := image.Rect(offsetX, offsetY, offsetX+gw, offsetY+gh)
		// copy glyph alpha into atlas
		draw.Draw(atlasImg, dstRect, mask, maskp, draw.Src)

		fc := FontCharacter{
			AtlasX:   float32(offsetX),
			AtlasY:   float32(offsetY),
			Width:    float32(gw),
			Height:   float32(gh),
			BearingX: float32(dr.Min.X),
			BearingY: float32(-dr.Min.Y),
			Advance:  int(math.Round(float64(advance) / 64.0)),
		}
		characters[r] = fc

		offsetX += gw + padding
		if gh > rowHeight {
			rowHeight = gh
		}
	}

	// Upload atlas to OpenGL as GL_RED
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	// Ensure tight byte alignment for single-channel (alpha) upload
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, int32(atlasW), int32(atlasH), 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(atlasImg.Pix))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return &FontAtlasInfo{TextureID: texture, AtlasW: atlasW, AtlasH: atlasH, Characters: characters}, nil
}

// FontRenderer renders ASCII text strings using a prebuilt atlas
type FontRenderer struct {
	atlas       *FontAtlasInfo
	shader      *Shader
	projection  mgl32.Mat4
	vao         uint32
	vbo         uint32
	maxCharsCap int
}

// NewFontRenderer creates the renderer and loads the font shader from assets
func NewFontRenderer(atlas *FontAtlasInfo) (*FontRenderer, error) {
	if atlas == nil || len(atlas.Characters) == 0 {
		return nil, fmt.Errorf("invalid font atlas")
	}
	vert := filepath.Join(ShadersDir, "font.vert")
	frag := filepath.Join(ShadersDir, "font.frag")
	shader, err := NewShader(vert, frag)
	if err != nil {
		return nil, err
	}
	fr := &FontRenderer{
		atlas:       atlas,
		shader:      shader,
		maxCharsCap: 256,
		projection:  mgl32.Ortho(0, float32(WinWidth), float32(WinHeight), 0, 0, 1),
	}
	fr.initGL()
	return fr, nil
}

func (fr *FontRenderer) initGL() {
	gl.GenVertexArrays(1, &fr.vao)
	gl.GenBuffers(1, &fr.vbo)
	gl.BindVertexArray(fr.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, fr.vbo)
	// Allocate a dynamic buffer for up to maxCharsCap characters (6 verts per char, 4 floats per vert)
	capFloats := fr.maxCharsCap * 6 * 4
	gl.BufferData(gl.ARRAY_BUFFER, capFloats*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 4, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}

// Render draws the given text at (x,y) using the provided projection matrix and RGB color.
// x and y are in the same coordinate system as the projection matrix expects (e.g., pixels in an orthographic projection).
func (fr *FontRenderer) Render(text string, x, y, scale float32, color mgl32.Vec3) {
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)

	fr.shader.Use()
	fr.shader.SetVector3("textColor", color.X(), color.Y(), color.Z())
	fr.shader.SetMatrix4("projection", &fr.projection[0])
	fr.shader.SetInt("text", 0)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, fr.atlas.TextureID)
	gl.BindVertexArray(fr.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, fr.vbo)

	// Build vertex data for all characters
	verts := fr.buildVertices([]rune(text), x, y, scale)
	// Resize buffer if needed
	needed := len(verts) * 4
	if needed > fr.maxCharsCap*6*4 {
		fr.maxCharsCap = int(math.Max(float64(fr.maxCharsCap*2), float64(len(verts)/6+1)))
		gl.BufferData(gl.ARRAY_BUFFER, fr.maxCharsCap*6*4*4, nil, gl.DYNAMIC_DRAW)
	}
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(verts)*4, gl.Ptr(verts))
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(verts)/4))

	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
}

func (fr *FontRenderer) buildVertices(chars []rune, x, y, scale float32) []float32 {
	vertices := make([]float32, 0, len(chars)*6*4)
	for _, r := range chars {
		fc, ok := fr.atlas.Characters[r]
		if !ok {
			// Skip missing glyphs
			x += float32(fr.atlas.Characters[' '].Advance) * scale
			continue
		}
		quad := fr.buildCharVertices(fc, x, y, scale)
		vertices = append(vertices, quad...)
		x += float32(fc.Advance) * scale
	}
	return vertices
}

func (fr *FontRenderer) buildCharVertices(fc FontCharacter, x, y, scale float32) []float32 {
	// Screen position
	xPos := x + fc.BearingX*scale
	yPos := y - fc.BearingY*scale
	w := fc.Width * scale
	h := fc.Height * scale

	// Texture coordinates (normalized)
	atlasX := fc.AtlasX / float32(fr.atlas.AtlasW)
	atlasY := fc.AtlasY / float32(fr.atlas.AtlasH)
	wA := fc.Width / float32(fr.atlas.AtlasW)
	hA := fc.Height / float32(fr.atlas.AtlasH)

	return []float32{
		// triangle 1
		xPos, yPos + h, atlasX, atlasY + hA,
		xPos, yPos, atlasX, atlasY,
		xPos + w, yPos, atlasX + wA, atlasY,
		// triangle 2
		xPos, yPos + h, atlasX, atlasY + hA,
		xPos + w, yPos, atlasX + wA, atlasY,
		xPos + w, yPos + h, atlasX + wA, atlasY + hA,
	}
}
