package items

import (
	"mini-mc/internal/registry"
	"mini-mc/pkg/blockmodel"
	"path/filepath"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type ItemMesh struct {
	VAO         uint32
	VBO         uint32
	VertexCount int32
}

const (
	// Position(3) + UV(2) + Normal(3) + TexID(1) + TintIndex(1)
	VertexSize = 10
	FloatSize  = 4
)

func BuildItemMesh(elements []blockmodel.Element) (*ItemMesh, error) {
	var vertices []float32

	for _, el := range elements {
		// Prepare rotation matrix
		var rotMat mgl32.Mat4
		hasRotation := el.Rotation != nil
		if hasRotation {
			rotMat = mgl32.Ident4()
			// Origin is in 0..16 space. Normalize to 0..1
			ox := el.Rotation.Origin[0] / 16.0
			oy := el.Rotation.Origin[1] / 16.0
			oz := el.Rotation.Origin[2] / 16.0

			rotMat = rotMat.Mul4(mgl32.Translate3D(ox, oy, oz))
			angleRad := mgl32.DegToRad(el.Rotation.Angle)
			switch el.Rotation.Axis {
			case "x":
				rotMat = rotMat.Mul4(mgl32.HomogRotate3DX(angleRad))
			case "y":
				rotMat = rotMat.Mul4(mgl32.HomogRotate3DY(angleRad))
			case "z":
				rotMat = rotMat.Mul4(mgl32.HomogRotate3DZ(angleRad))
			}
			rotMat = rotMat.Mul4(mgl32.Translate3D(-ox, -oy, -oz))
		}

		// Normalize coordinates to 0..1
		x1, y1, z1 := el.From[0]/16.0, el.From[1]/16.0, el.From[2]/16.0
		x2, y2, z2 := el.To[0]/16.0, el.To[1]/16.0, el.To[2]/16.0

		for faceName, faceDef := range el.Faces {
			texID := float32(0)

			// Resolve texture name to ID
			// Typically faceDef.Texture is like "blocks/grass_top"
			// But TextureMap keys are "grass_top.png"
			texName := faceDef.Texture
			if !strings.HasSuffix(texName, ".png") {
				base := filepath.Base(texName)
				if base != "." && base != "/" {
					texName = base + ".png"
				}
			}

			if id, ok := registry.TextureMap[texName]; ok {
				texID = float32(id)
			} else {
				// Fallback to try original name if not found
				if id, ok := registry.TextureMap[faceDef.Texture]; ok {
					texID = float32(id)
				}
			}

			tintIdx := float32(-1.0)
			if faceDef.TintIndex != nil {
				tintIdx = float32(*faceDef.TintIndex)
			}

			// Define quad vertices based on face direction
			// Using standard Minecraft UV convention if UV is nil, else use provided UV
			var quad [4][8]float32 // Pos(3) + UV(2) + Normal(3)

			// Normal
			var nx, ny, nz float32

			// UVs: Minecraft default is usually based on position. blockmodel loader might provide default UVs?
			// If face.UV is all 0, we should infer. But usually loader or JSON has it.
			// faceDef.UV is [4]float32 {x1, y1, x2, y2}
			u1, v1, u2, v2 := faceDef.UV[0]/16.0, faceDef.UV[1]/16.0, faceDef.UV[2]/16.0, faceDef.UV[3]/16.0

			switch faceName {
			case "north": // -Z
				nx, ny, nz = 0, 0, -1
				quad[0] = [8]float32{x2, y2, z1, u1, v1, nx, ny, nz} // Top Right
				quad[1] = [8]float32{x2, y1, z1, u1, v2, nx, ny, nz} // Bot Right
				quad[2] = [8]float32{x1, y1, z1, u2, v2, nx, ny, nz} // Bot Left
				quad[3] = [8]float32{x1, y2, z1, u2, v1, nx, ny, nz} // Top Left
			case "south": // +Z
				nx, ny, nz = 0, 0, 1
				quad[0] = [8]float32{x1, y2, z2, u1, v1, nx, ny, nz}
				quad[1] = [8]float32{x1, y1, z2, u1, v2, nx, ny, nz}
				quad[2] = [8]float32{x2, y1, z2, u2, v2, nx, ny, nz}
				quad[3] = [8]float32{x2, y2, z2, u2, v1, nx, ny, nz}
			case "west": // -X
				nx, ny, nz = -1, 0, 0
				quad[0] = [8]float32{x1, y2, z1, u1, v1, nx, ny, nz}
				quad[1] = [8]float32{x1, y1, z1, u1, v2, nx, ny, nz}
				quad[2] = [8]float32{x1, y1, z2, u2, v2, nx, ny, nz}
				quad[3] = [8]float32{x1, y2, z2, u2, v1, nx, ny, nz}
			case "east": // +X
				nx, ny, nz = 1, 0, 0
				quad[0] = [8]float32{x2, y2, z2, u1, v1, nx, ny, nz}
				quad[1] = [8]float32{x2, y1, z2, u1, v2, nx, ny, nz}
				quad[2] = [8]float32{x2, y1, z1, u2, v2, nx, ny, nz}
				quad[3] = [8]float32{x2, y2, z1, u2, v1, nx, ny, nz}
			case "up": // +Y
				nx, ny, nz = 0, 1, 0
				quad[0] = [8]float32{x1, y2, z1, u1, v1, nx, ny, nz}
				quad[1] = [8]float32{x1, y2, z2, u1, v2, nx, ny, nz}
				quad[2] = [8]float32{x2, y2, z2, u2, v2, nx, ny, nz}
				quad[3] = [8]float32{x2, y2, z1, u2, v1, nx, ny, nz}
			case "down": // -Y
				nx, ny, nz = 0, -1, 0
				quad[0] = [8]float32{x1, y1, z2, u1, v1, nx, ny, nz}
				quad[1] = [8]float32{x1, y1, z1, u1, v2, nx, ny, nz}
				quad[2] = [8]float32{x2, y1, z1, u2, v2, nx, ny, nz}
				quad[3] = [8]float32{x2, y1, z2, u2, v1, nx, ny, nz}
			}

			// Apply rotation and append as 2 triangles
			indices := []int{0, 1, 2, 0, 2, 3}
			for _, idx := range indices {
				v := quad[idx]
				pos := mgl32.Vec4{v[0], v[1], v[2], 1.0}
				if hasRotation {
					pos = rotMat.Mul4x1(pos)
				}

				// Normal rotation
				norm := mgl32.Vec4{v[5], v[6], v[7], 0.0}
				if hasRotation {
					norm = rotMat.Mul4x1(norm)
				}

				// Append
				vertices = append(vertices,
					pos.X(), pos.Y(), pos.Z(),
					v[3], v[4],
					norm.X(), norm.Y(), norm.Z(),
					texID,
					tintIdx,
				)
			}
		}
	}

	if len(vertices) == 0 {
		return nil, nil // Empty mesh
	}

	vao := uint32(0)
	vbo := uint32(0)
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*FloatSize, gl.Ptr(vertices), gl.STATIC_DRAW)

	stride := int32(VertexSize * FloatSize)
	// Pos
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	// UV
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*FloatSize))
	// Normal
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(5*FloatSize))
	// TexID
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(3, 1, gl.FLOAT, false, stride, gl.PtrOffset(8*FloatSize))
	// TintIndex
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointer(4, 1, gl.FLOAT, false, stride, gl.PtrOffset(9*FloatSize))

	gl.BindVertexArray(0)

	return &ItemMesh{
		VAO:         vao,
		VBO:         vbo,
		VertexCount: int32(len(vertices) / VertexSize),
	}, nil
}
