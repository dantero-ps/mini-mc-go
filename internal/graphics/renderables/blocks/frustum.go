package blocks

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// Frustum culling margin in blocks (inflates AABBs before testing)
var frustumMargin float32 = 1.0

// extractFrustumPlanes builds six planes from the combined projection*view matrix.
// Planes are returned in order: left, right, bottom, top, near, far.
func extractFrustumPlanes(clip mgl32.Mat4) [6]plane {
	// Matrix is in column-major order in mgl32
	m00, m01, m02, m03 := clip[0], clip[4], clip[8], clip[12]
	m10, m11, m12, m13 := clip[1], clip[5], clip[9], clip[13]
	m20, m21, m22, m23 := clip[2], clip[6], clip[10], clip[14]
	m30, m31, m32, m33 := clip[3], clip[7], clip[11], clip[15]

	pl := [6]plane{}
	// Left  = m3 + m0
	pl[0] = normalizePlane(plane{m30 + m00, m31 + m01, m32 + m02, m33 + m03})
	// Right = m3 - m0
	pl[1] = normalizePlane(plane{m30 - m00, m31 - m01, m32 - m02, m33 - m03})
	// Bottom = m3 + m1
	pl[2] = normalizePlane(plane{m30 + m10, m31 + m11, m32 + m12, m33 + m13})
	// Top = m3 - m1
	pl[3] = normalizePlane(plane{m30 - m10, m31 - m11, m32 - m12, m33 - m13})
	// Near = m3 + m2
	pl[4] = normalizePlane(plane{m30 + m20, m31 + m21, m32 + m22, m33 + m23})
	// Far = m3 - m2
	pl[5] = normalizePlane(plane{m30 - m20, m31 - m21, m32 - m22, m33 - m23})
	return pl
}

func normalizePlane(p plane) plane {
	len := float32(math.Sqrt(float64(p.a*p.a + p.b*p.b + p.c*p.c)))
	if len == 0 {
		return p
	}
	return plane{p.a / len, p.b / len, p.c / len, p.d / len}
}

// aabbIntersectsFrustumPlanes tests AABB against precomputed planes.
func aabbIntersectsFrustumPlanes(min, max mgl32.Vec3, planes [6]plane) bool {
	for i := 0; i < 6; i++ {
		p := planes[i]
		// Select the positive vertex for this plane normal
		px := max.X()
		if p.a < 0 {
			px = min.X()
		}
		py := max.Y()
		if p.b < 0 {
			py = min.Y()
		}
		pz := max.Z()
		if p.c < 0 {
			pz = min.Z()
		}
		// If positive vertex is outside, AABB is outside
		if p.a*px+p.b*py+p.c*pz+p.d < 0 {
			return false
		}
	}
	return true
}

// aabbIntersectsFrustumPlanesF is a float-optimized variant to avoid Vec3 calls in hot paths
func aabbIntersectsFrustumPlanesF(minx, miny, minz, maxx, maxy, maxz float32, planes [6]plane) bool {
	// Unrolled loop for better performance - frustum culling is called very frequently
	p := planes[0] // left
	px := maxx
	if p.a < 0 {
		px = minx
	}
	py := maxy
	if p.b < 0 {
		py = miny
	}
	pz := maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[1] // right
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[2] // bottom
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[3] // top
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[4] // near
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[5] // far
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	return true
}

// matrixNearEqual compares two matrices for approximate equality within epsilon
func matrixNearEqual(a, b mgl32.Mat4, epsilon float32) bool {
	for i := 0; i < 16; i++ {
		if float32(math.Abs(float64(a[i]-b[i]))) > epsilon {
			return false
		}
	}
	return true
}
