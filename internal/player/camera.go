package player

import (
	"math"
	"mini-mc/internal/config"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func (p *Player) HandleMouseMovement(w *glfw.Window, xpos, ypos float64) {
	if p.FirstMouse {
		p.LastMouseX = xpos
		p.LastMouseY = ypos
		p.FirstMouse = false
		return
	}

	xoffset := xpos - p.LastMouseX
	yoffset := p.LastMouseY - ypos
	p.LastMouseX = xpos
	p.LastMouseY = ypos

	sensitivity := 0.1
	xoffset *= sensitivity
	yoffset *= sensitivity

	p.CamYaw += xoffset
	p.CamPitch += yoffset

	// Constrain pitch
	if p.CamPitch > 89.0 {
		p.CamPitch = 89.0
	}
	if p.CamPitch < -89.0 {
		p.CamPitch = -89.0
	}
}

func (p *Player) UpdateHeadBob() {
	p.PrevHeadBobYaw = p.HeadBobYaw
	p.PrevHeadBobPitch = p.HeadBobPitch

	f := math.Sqrt(float64(p.Velocity.X()*p.Velocity.X() + p.Velocity.Z()*p.Velocity.Z()))
	f1 := math.Atan(float64(-p.Velocity.Y()*0.20000000298023224)) * 15.0

	if f > 0.1 {
		f = 0.1
	}

	// Smooth transition: when in air, gradually reduce instead of instantly setting to 0
	if !p.OnGround {
		blendFactor := 0.1
		p.HeadBobYaw += (0.0 - p.HeadBobYaw) * blendFactor
	} else {
		p.HeadBobYaw += (f - p.HeadBobYaw) * 0.1
	}

	if p.OnGround {
		f1 = 0.0
		p.HeadBobPitch += (f1 - p.HeadBobPitch) * 0.3
	} else {
		p.HeadBobPitch += (f1 - p.HeadBobPitch) * 0.3
	}
}

// UpdateCameraBob updates cameraYaw and cameraPitch for view bobbing animation
func (p *Player) UpdateCameraBob() {
	p.PrevCameraYaw = p.CameraYaw
	p.PrevCameraPitch = p.CameraPitch

	if !config.GetViewBobbing() {
		p.CameraYaw += (0.0 - p.CameraYaw) * 0.1
		p.CameraPitch += (0.0 - p.CameraPitch) * 0.1
		return
	}

	f := float32(math.Sqrt(float64(p.Velocity.X()*p.Velocity.X() + p.Velocity.Z()*p.Velocity.Z())))
	f1 := float32(math.Atan(float64(-p.Velocity.Y()*0.20000000298023224)) * 1.5)

	if f > 0.1 {
		f = 0.1
	}

	if !p.OnGround {
		f = 0.0
	}

	if p.OnGround {
		f1 = 0.0
	}

	p.CameraYaw += (f - p.CameraYaw) * 0.03
	p.CameraPitch += (f1 - p.CameraPitch) * 0.1
}

// UpdateRenderArm updates renderArmYaw and renderArmPitch for hand sway animation
func (p *Player) UpdateRenderArm(dt float64) {
	p.PrevRenderArmYaw = p.RenderArmYaw
	p.PrevRenderArmPitch = p.RenderArmPitch

	decaySpeed := 25.0
	factor := float32(1.0 - math.Exp(-decaySpeed*dt))

	p.RenderArmPitch += (float32(p.CamPitch) - p.RenderArmPitch) * factor
	p.RenderArmYaw += (float32(p.CamYaw) - p.RenderArmYaw) * factor
}

func (p *Player) GetFrontVector() mgl32.Vec3 {
	y := mgl32.DegToRad(float32(p.CamYaw))
	pt := mgl32.DegToRad(float32(p.CamPitch))
	fx := float32(math.Cos(float64(y)) * math.Cos(float64(pt)))
	fy := float32(math.Sin(float64(pt)))
	fz := float32(math.Sin(float64(y)) * math.Cos(float64(pt)))
	return mgl32.Vec3{fx, fy, fz}.Normalize()
}

func (p *Player) GetViewMatrix() mgl32.Mat4 {
	return p.GetViewMatrixWithPartialTicks(0.0)
}

func (p *Player) GetViewMatrixWithPartialTicks(partialTicks float32) mgl32.Mat4 {
	eyePos := p.GetEyePosition()
	front := p.GetFrontVector()
	target := eyePos.Add(front)

	viewMatrix := mgl32.LookAtV(eyePos, target, mgl32.Vec3{0, 1, 0})

	if !config.GetViewBobbing() {
		return viewMatrix
	}

	f := float32(p.DistanceWalkedModified - p.PrevDistanceWalkedModified)
	f1 := float32(p.DistanceWalkedModified + float64(f)*float64(partialTicks))
	f2 := p.PrevCameraYaw + (p.CameraYaw-p.PrevCameraYaw)*partialTicks
	f3 := p.PrevCameraPitch + (p.CameraPitch-p.PrevCameraPitch)*partialTicks

	translateX := float32(math.Sin(float64(f1*math.Pi))) * f2 * 0.5
	translateY := -float32(math.Abs(math.Cos(float64(f1*math.Pi))) * float64(f2))
	rotateZ := float32(math.Sin(float64(f1*math.Pi))) * f2 * 3.0
	rotateX := float32(math.Abs(math.Cos(float64(f1*math.Pi-0.2))*float64(f2))) * 5.0

	translateMat := mgl32.Translate3D(translateX, translateY, 0.0)
	rotateZMat := mgl32.HomogRotate3D(mgl32.DegToRad(rotateZ), mgl32.Vec3{0, 0, 1})
	rotateXMat := mgl32.HomogRotate3D(mgl32.DegToRad(rotateX), mgl32.Vec3{1, 0, 0})
	cameraPitchMat := mgl32.HomogRotate3D(mgl32.DegToRad(f3), mgl32.Vec3{1, 0, 0})

	bobbingMat := translateMat.Mul4(rotateZMat).Mul4(rotateXMat).Mul4(cameraPitchMat)

	return bobbingMat.Mul4(viewMatrix)
}

// TriggerHandSwing starts a new right-hand swing animation.
func (p *Player) TriggerHandSwing() {
	p.handSwingTimer = p.handSwingDuration
}

// GetHandSwingProgress returns current right-hand swing progress in [0,1].
func (p *Player) GetHandSwingProgress() float32 {
	return p.HandSwingProgress
}

// GetHandEquipProgress returns current equip animation progress in [0,1].
func (p *Player) GetHandEquipProgress() float32 {
	return p.EquipProgress
}

func (p *Player) updateEquippedItem(dt float32) {
	itemstack := p.Inventory.GetCurrentItem()
	flag := false

	if p.EquippedItem != nil && itemstack != nil {
		if p.EquippedItem.Type != itemstack.Type {
			flag = true
		}
	} else if p.EquippedItem == nil && itemstack == nil {
		flag = false
	} else {
		flag = true
	}

	f := float32(0.4 * 20.0 * dt)
	f1 := float32(0.0)
	if !flag {
		f1 = 1.0
	}

	f2 := f1 - p.EquipProgress
	if f2 < -f {
		f2 = -f
	}
	if f2 > f {
		f2 = f
	}

	p.EquipProgress += f2

	if p.EquipProgress < 0.1 {
		p.EquippedItem = itemstack
	}
}
