package main

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func init() { runtime.LockOSThread() }

const (
	winW = 900
	winH = 600

	// Block interaction distance range
	minReachDistance = 0.1
	maxReachDistance = 5.0

	// World bounds for 3D array: x/z -8 to 8 (17), y 0 to 16 (17)
	worldSizeX   = 17
	worldSizeY   = 17
	worldSizeZ   = 17
	worldOffsetX = 8
	worldOffsetY = 0
	worldOffsetZ = 8
)

var cubeVertices = []float32{
	-0.5, -0.5, 0.5, 0, 0, 1,
	0.5, -0.5, 0.5, 0, 0, 1,
	0.5, 0.5, 0.5, 0, 0, 1,
	0.5, 0.5, 0.5, 0, 0, 1,
	-0.5, 0.5, 0.5, 0, 0, 1,
	-0.5, -0.5, 0.5, 0, 0, 1,
	0.5, -0.5, -0.5, 0, 0, -1,
	-0.5, -0.5, -0.5, 0, 0, -1,
	-0.5, 0.5, -0.5, 0, 0, -1,
	-0.5, 0.5, -0.5, 0, 0, -1,
	0.5, 0.5, -0.5, 0, 0, -1,
	0.5, -0.5, -0.5, 0, 0, -1,
	-0.5, -0.5, -0.5, -1, 0, 0,
	-0.5, -0.5, 0.5, -1, 0, 0,
	-0.5, 0.5, 0.5, -1, 0, 0,
	-0.5, 0.5, 0.5, -1, 0, 0,
	-0.5, 0.5, -0.5, -1, 0, 0,
	-0.5, -0.5, -0.5, -1, 0, 0,
	0.5, -0.5, 0.5, 1, 0, 0,
	0.5, -0.5, -0.5, 1, 0, 0,
	0.5, 0.5, -0.5, 1, 0, 0,
	0.5, 0.5, -0.5, 1, 0, 0,
	0.5, 0.5, 0.5, 1, 0, 0,
	0.5, -0.5, 0.5, 1, 0, 0,
	-0.5, 0.5, 0.5, 0, 1, 0,
	0.5, 0.5, 0.5, 0, 1, 0,
	0.5, 0.5, -0.5, 0, 1, 0,
	0.5, 0.5, -0.5, 0, 1, 0,
	-0.5, 0.5, -0.5, 0, 1, 0,
	-0.5, 0.5, 0.5, 0, 1, 0,
	-0.5, -0.5, -0.5, 0, -1, 0,
	0.5, -0.5, -0.5, 0, -1, 0,
	0.5, -0.5, 0.5, 0, -1, 0,
	0.5, -0.5, 0.5, 0, -1, 0,
	-0.5, -0.5, 0.5, 0, -1, 0,
	-0.5, -0.5, -0.5, 0, -1, 0,
}

// Wireframe cube edges for highlighting
var cubeWireframeVertices = []float32{
	-0.5, -0.5, -0.5, 0.5, -0.5, -0.5,
	0.5, -0.5, -0.5, 0.5, -0.5, 0.5,
	0.5, -0.5, 0.5, -0.5, -0.5, 0.5,
	-0.5, -0.5, 0.5, -0.5, -0.5, -0.5,
	-0.5, 0.5, -0.5, 0.5, 0.5, -0.5,
	0.5, 0.5, -0.5, 0.5, 0.5, 0.5,
	0.5, 0.5, 0.5, -0.5, 0.5, 0.5,
	-0.5, 0.5, 0.5, -0.5, 0.5, -0.5,
	-0.5, -0.5, -0.5, -0.5, 0.5, -0.5,
	0.5, -0.5, -0.5, 0.5, 0.5, -0.5,
	0.5, -0.5, 0.5, 0.5, 0.5, 0.5,
	-0.5, -0.5, 0.5, -0.5, 0.5, 0.5,
}

// Crosshair vertices
var crosshairVertices = []float32{
	-0.02, 0.0,
	0.02, 0.0,
	0.0, -0.02,
	0.0, 0.02,
}

var vertexShader = `#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
layout(location = 2) in vec3 instancePos;
uniform mat4 view;
uniform mat4 proj;
out vec3 Normal;
out vec3 FragPos;
void main() {
	vec3 pos = aPos + instancePos;
	FragPos = pos;
	Normal = aNormal;
	gl_Position = proj * view * vec4(pos, 1.0);
}
`

var fragmentShader = `#version 330 core
in vec3 Normal;
in vec3 FragPos;
uniform vec3 color;
uniform vec3 lightDir;
out vec4 FragColor;
void main() {
	vec3 n = normalize(Normal);
	float diff = max(dot(n, -lightDir), 0.3);
	float edge = pow(1.0 - max(dot(n, vec3(0,1,0)), 0.0), 3.0);
	vec3 col = color * diff * (1.0 - 0.2 * edge);
	FragColor = vec4(col, 1.0);
}
`

// Simple shader for wireframe and 2D elements
var simpleVertexShader = `#version 330 core
layout(location = 0) in vec3 aPos;
uniform mat4 model;
uniform mat4 view;
uniform mat4 proj;
void main() {
	gl_Position = proj * view * model * vec4(aPos, 1.0);
}
`

var simpleFragmentShader = `#version 330 core
uniform vec3 color;
out vec4 FragColor;
void main() {
	FragColor = vec4(color, 1.0);
}
`

// 2D shader for crosshair
var crosshairVertexShader = `#version 330 core
layout(location = 0) in vec2 aPos;
void main() {
	gl_Position = vec4(aPos, 0.0, 1.0);
}
`

var crosshairFragmentShader = `#version 330 core
out vec4 FragColor;
void main() {
	FragColor = vec4(1.0, 1.0, 1.0, 0.8);
}
`

var (
	playerPos               = mgl32.Vec3{0, 2.8, 0}
	velocityY       float32 = 0
	gravity         float32 = 15.0
	jumpSpeed       float32 = 6.0
	onGround        bool    = false
	hoveredBlock    [3]int  // Currently hovered block for highlighting
	hasHoveredBlock bool    // Whether there's a block being hovered
)

type World struct {
	blocks [worldSizeX][worldSizeY][worldSizeZ]bool
}

func (w *World) Get(x, y, z int) bool {
	ix, iy, iz := w.toIndex(x, y, z)
	if ix < 0 || ix >= worldSizeX || iy < 0 || iy >= worldSizeY || iz < 0 || iz >= worldSizeZ {
		return false
	}
	return w.blocks[ix][iy][iz]
}

func (w *World) Set(x, y, z int, val bool) {
	ix, iy, iz := w.toIndex(x, y, z)
	if ix < 0 || ix >= worldSizeX || iy < 0 || iy >= worldSizeY || iz < 0 || iz >= worldSizeZ {
		return
	}
	w.blocks[ix][iy][iz] = val
}

func (w *World) toIndex(x, y, z int) (int, int, int) {
	return x + worldOffsetX, y + worldOffsetY, z + worldOffsetZ
}

func (w *World) fromIndex(ix, iy, iz int) (int, int, int) {
	return ix - worldOffsetX, iy - worldOffsetY, iz - worldOffsetZ
}

func (w *World) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3
	for ix := 0; ix < worldSizeX; ix++ {
		for iy := 0; iy < worldSizeY; iy++ {
			for iz := 0; iz < worldSizeZ; iz++ {
				if w.blocks[ix][iy][iz] {
					x, y, z := w.fromIndex(ix, iy, iz)
					positions = append(positions, mgl32.Vec3{float32(x), float32(y), float32(z)})
				}
			}
		}
	}
	return positions
}

// DDA-based raycast for efficiency
func raycast(start mgl32.Vec3, direction mgl32.Vec3, minDist, maxDist float32, world *World) ([3]int, [3]int, float32, bool) {
	pos := start
	dir := direction.Normalize()

	gridX := int(math.Floor(float64(pos.X())))
	gridY := int(math.Floor(float64(pos.Y())))
	gridZ := int(math.Floor(float64(pos.Z())))

	deltaX := float32(math.Abs(1.0 / float64(dir.X())))
	deltaY := float32(math.Abs(1.0 / float64(dir.Y())))
	deltaZ := float32(math.Abs(1.0 / float64(dir.Z())))

	var stepX, stepY, stepZ int
	var sideDistX, sideDistY, sideDistZ float32

	if dir.X() > 0 {
		stepX = 1
		sideDistX = (float32(gridX) + 1 - pos.X()) * deltaX
	} else {
		stepX = -1
		sideDistX = (pos.X() - float32(gridX)) * deltaX
	}
	if dir.Y() > 0 {
		stepY = 1
		sideDistY = (float32(gridY) + 1 - pos.Y()) * deltaY
	} else {
		stepY = -1
		sideDistY = (pos.Y() - float32(gridY)) * deltaY
	}
	if dir.Z() > 0 {
		stepZ = 1
		sideDistZ = (float32(gridZ) + 1 - pos.Z()) * deltaZ
	} else {
		stepZ = -1
		sideDistZ = (pos.Z() - float32(gridZ)) * deltaZ
	}

	var lastEmptyPos [3]int = [3]int{gridX, gridY, gridZ}
	var hitDistance float32

	for hitDistance < maxDist {
		if sideDistX < sideDistY && sideDistX < sideDistZ {
			sideDistX += deltaX
			gridX += stepX
			hitDistance = sideDistX - deltaX
		} else if sideDistY < sideDistZ {
			sideDistY += deltaY
			gridY += stepY
			hitDistance = sideDistY - deltaY
		} else {
			sideDistZ += deltaZ
			gridZ += stepZ
			hitDistance = sideDistZ - deltaZ
		}

		if hitDistance < minDist {
			continue
		}

		blockPos := [3]int{gridX, gridY, gridZ}
		if world.Get(blockPos[0], blockPos[1], blockPos[2]) {
			return blockPos, lastEmptyPos, hitDistance, true
		}

		lastEmptyPos = blockPos
	}

	return [3]int{}, [3]int{}, 0, false
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	window, err := glfw.CreateWindow(winW, winH, "mini-mc", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		panic(err)
	}

	glfw.SwapInterval(0) // Disable V-Sync

	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	simpleProgram, err := newProgram(simpleVertexShader, simpleFragmentShader)
	if err != nil {
		panic(err)
	}

	crosshairProgram, err := newProgram(crosshairVertexShader, crosshairFragmentShader)
	if err != nil {
		panic(err)
	}

	// Setup cube VAO for instancing
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	stride := int32(6 * 4)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(3*4))

	// Instance buffer
	var instanceVBO uint32
	gl.GenBuffers(1, &instanceVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, instanceVBO)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.VertexAttribDivisor(2, 1)

	// Setup wireframe VAO
	var wireframeVAO uint32
	gl.GenVertexArrays(1, &wireframeVAO)
	gl.BindVertexArray(wireframeVAO)

	var wireframeVBO uint32
	gl.GenBuffers(1, &wireframeVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, wireframeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeWireframeVertices)*4, gl.Ptr(cubeWireframeVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))

	// Setup crosshair VAO
	var crosshairVAO uint32
	gl.GenVertexArrays(1, &crosshairVAO)
	gl.BindVertexArray(crosshairVAO)

	var crosshairVBO uint32
	gl.GenBuffers(1, &crosshairVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, crosshairVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(crosshairVertices)*4, gl.Ptr(crosshairVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))

	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	camYaw := -90.0
	camPitch := -20.0
	lastX := float64(winW) / 2
	lastY := float64(winH) / 2
	firstMouse := true
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	window.SetCursorPosCallback(func(w *glfw.Window, xpos float64, ypos float64) {
		if firstMouse {
			lastX = xpos
			lastY = ypos
			firstMouse = false
		}
		xoffset := xpos - lastX
		yoffset := lastY - ypos
		lastX = xpos
		lastY = ypos
		sensitivity := 0.1
		xoffset *= sensitivity
		yoffset *= sensitivity
		camYaw += xoffset
		camPitch += yoffset
		if camPitch > 89 {
			camPitch = 89
		}
		if camPitch < -89 {
			camPitch = -89
		}
	})

	world := &World{}
	for x := -8; x <= 8; x++ {
		for z := -8; z <= 8; z++ {
			world.Set(x, 0, z, true)
		}
	}

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press && hasHoveredBlock {
			if button == glfw.MouseButtonLeft {
				// Break block
				world.Set(hoveredBlock[0], hoveredBlock[1], hoveredBlock[2], false)
			}
			if button == glfw.MouseButtonRight {
				// Place block at the adjacent position
				front := getFrontVector(camYaw, camPitch)
				rayStart := playerPos.Add(mgl32.Vec3{0, 1.5, 0})
				_, placePos, _, hit := raycast(rayStart, front, minReachDistance, maxReachDistance, world)
				if hit {
					world.Set(placePos[0], placePos[1], placePos[2], true)
				}
			}
		}
	})

	lastTime := time.Now()

	for !window.ShouldClose() {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		updatePlayerPosition(dt, world, camYaw, camPitch, window)
		updateHoveredBlock(camYaw, camPitch, world)

		gl.ClearColor(0.53, 0.81, 0.92, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Render blocks with instancing
		gl.UseProgram(program)

		proj := mgl32.Perspective(mgl32.DegToRad(60.0), float32(winW)/float32(winH), 0.1, 100.0)
		front := getFrontVector(camYaw, camPitch)
		eyePos := playerPos.Add(mgl32.Vec3{0, 1.5, 0})
		view := mgl32.LookAtV(eyePos, eyePos.Add(front), mgl32.Vec3{0, 1, 0})

		projLoc := gl.GetUniformLocation(program, gl.Str("proj\x00"))
		viewLoc := gl.GetUniformLocation(program, gl.Str("view\x00"))
		colorLoc := gl.GetUniformLocation(program, gl.Str("color\x00"))
		lightLoc := gl.GetUniformLocation(program, gl.Str("lightDir\x00"))

		gl.UniformMatrix4fv(projLoc, 1, false, &proj[0])
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

		light := mgl32.Vec3{0.5, 1.0, 0.3}.Normalize()
		gl.Uniform3f(lightLoc, light.X(), light.Y(), light.Z())

		// For simplicity, all blocks same color; adjust if needed
		gl.Uniform3f(colorLoc, 0.3, 0.9, 0.3) // Default grass-like

		gl.BindVertexArray(vao)
		positions := world.GetActiveBlocks()
		if len(positions) > 0 {
			gl.BindBuffer(gl.ARRAY_BUFFER, instanceVBO)
			gl.BufferData(gl.ARRAY_BUFFER, len(positions)*3*4, gl.Ptr(positions), gl.DYNAMIC_DRAW)
			gl.DrawArraysInstanced(gl.TRIANGLES, 0, int32(len(cubeVertices)/6), int32(len(positions)))
		}

		// Render block outline for hovered block
		if hasHoveredBlock {
			gl.UseProgram(simpleProgram)

			simpleProjLoc := gl.GetUniformLocation(simpleProgram, gl.Str("proj\x00"))
			simpleViewLoc := gl.GetUniformLocation(simpleProgram, gl.Str("view\x00"))
			simpleModelLoc := gl.GetUniformLocation(simpleProgram, gl.Str("model\x00"))
			simpleColorLoc := gl.GetUniformLocation(simpleProgram, gl.Str("color\x00"))

			gl.UniformMatrix4fv(simpleProjLoc, 1, false, &proj[0])
			gl.UniformMatrix4fv(simpleViewLoc, 1, false, &view[0])

			model := mgl32.Translate3D(float32(hoveredBlock[0]), float32(hoveredBlock[1]), float32(hoveredBlock[2])).Mul4(mgl32.Scale3D(1.01, 1.01, 1.01))
			gl.UniformMatrix4fv(simpleModelLoc, 1, false, &model[0])
			gl.Uniform3f(simpleColorLoc, 0.0, 0.0, 0.0) // Black outline

			gl.BindVertexArray(wireframeVAO)
			gl.LineWidth(2.0)
			gl.DrawArrays(gl.LINES, 0, int32(len(cubeWireframeVertices)/3))
		}

		// Render crosshair
		gl.UseProgram(crosshairProgram)
		gl.BindVertexArray(crosshairVAO)
		gl.LineWidth(2.0)
		gl.DrawArrays(gl.LINES, 0, 4)

		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func updateHoveredBlock(yaw, pitch float64, world *World) {
	front := getFrontVector(yaw, pitch)
	rayStart := playerPos.Add(mgl32.Vec3{0, 1.5, 0})
	hitPos, _, _, hit := raycast(rayStart, front, minReachDistance, maxReachDistance, world)

	if hit {
		hoveredBlock = hitPos
		hasHoveredBlock = true
	} else {
		hasHoveredBlock = false
	}
}

func updatePlayerPosition(dt float64, world *World, yaw, pitch float64, window *glfw.Window) {
	speed := float32(5.0)
	front := getFrontVector(yaw, pitch)
	right := front.Cross(mgl32.Vec3{0, 1, 0}).Normalize()

	moveDir := mgl32.Vec3{0, 0, 0}
	if window.GetKey(glfw.KeyW) == glfw.Press {
		moveDir = moveDir.Add(front)
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		moveDir = moveDir.Sub(front)
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		moveDir = moveDir.Sub(right)
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		moveDir = moveDir.Add(right)
	}
	if moveDir.Len() > 0 {
		moveDir = moveDir.Normalize().Mul(speed * float32(dt))
	}

	if onGround && window.GetKey(glfw.KeySpace) == glfw.Press {
		velocityY = jumpSpeed
		onGround = false
	}

	// Apply gravity
	velocityY -= gravity * float32(dt)
	moveY := velocityY * float32(dt)

	// Try horizontal movement first
	newPosH := playerPos.Add(mgl32.Vec3{moveDir.X(), 0, moveDir.Z()})
	if !collides(newPosH, world) {
		playerPos = newPosH
	}

	// Try vertical movement
	newPosV := playerPos.Add(mgl32.Vec3{0, moveY, 0})
	if !collides(newPosV, world) {
		playerPos = newPosV
		onGround = false
	} else {
		// Collision on vertical movement
		if velocityY < 0 {
			// Falling - find ground level
			groundY := findGroundLevel(playerPos.X(), playerPos.Z(), world)
			playerPos = mgl32.Vec3{playerPos.X(), groundY, playerPos.Z()}
			onGround = true
			velocityY = 0
		} else {
			// Hitting ceiling
			velocityY = 0
		}
	}
}

func findGroundLevel(x, z float32, world *World) float32 {
	// Consider player width
	minX := int(math.Floor(float64(x - 0.3)))
	maxX := int(math.Floor(float64(x + 0.3)))
	minZ := int(math.Floor(float64(z - 0.3)))
	maxZ := int(math.Floor(float64(z + 0.3)))

	maxGroundY := float32(-10)
	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			for by := int(math.Floor(float64(playerPos.Y()))); by >= -10; by-- {
				if world.Get(bx, by, bz) {
					groundY := float32(by) + 0.5 // Top of block
					if groundY > maxGroundY {
						maxGroundY = groundY
					}
					break
				}
			}
		}
	}
	if maxGroundY == -10 {
		return 1.0 // Safe default
	}
	return maxGroundY
}

func collides(pos mgl32.Vec3, world *World) bool {
	// Player AABB: width 0.6 (±0.3), height 1.8 (but using 2.5? Adjust to 1.8)
	minX := int(math.Floor(float64(pos.X() - 0.3)))
	maxX := int(math.Floor(float64(pos.X() + 0.3)))
	minY := int(math.Floor(float64(pos.Y())))
	maxY := int(math.Floor(float64(pos.Y() + 1.8)))
	minZ := int(math.Floor(float64(pos.Z() - 0.3)))
	maxZ := int(math.Floor(float64(pos.Z() + 0.3)))

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				if world.Get(x, y, z) {
					// Check actual AABB overlap
					blockMinX := float32(x) - 0.5
					blockMaxX := float32(x) + 0.5
					blockMinY := float32(y) - 0.5
					blockMaxY := float32(y) + 0.5
					blockMinZ := float32(z) - 0.5
					blockMaxZ := float32(z) + 0.5

					if pos.X()-0.3 < blockMaxX && pos.X()+0.3 > blockMinX &&
						pos.Y() < blockMaxY && pos.Y()+1.8 > blockMinY &&
						pos.Z()-0.3 < blockMaxZ && pos.Z()+0.3 > blockMinZ {
						return true
					}
				}
			}
		}
	}
	return false
}

func getFrontVector(yaw, pitch float64) mgl32.Vec3 {
	y := mgl32.DegToRad(float32(yaw))
	p := mgl32.DegToRad(float32(pitch))
	fx := float32(math.Cos(float64(y)) * math.Cos(float64(p)))
	fy := float32(math.Sin(float64(p)))
	fz := float32(math.Sin(float64(y)) * math.Cos(float64(p)))
	return mgl32.Vec3{fx, fy, fz}.Normalize()
}

func newProgram(vertexSrc, fragmentSrc string) (uint32, error) {
	vertexShader, err := compileShader(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}
