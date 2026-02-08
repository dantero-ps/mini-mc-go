#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aData; // Info(Normal+Bright), TexID, Tint

uniform mat4 view;
uniform mat4 proj;

out vec3 Normal;
out vec3 FragPos;
out vec3 TexCoord; // u, v, layer
out float Brightness;
out vec3 TintColor;

// Decode normal from encoded value
vec3 decodeNormal(int idx) {
	if (idx == 0) return vec3(0.0, 0.0, 1.0);   // North (+Z)
	if (idx == 1) return vec3(0.0, 0.0, -1.0);  // South (-Z)
	if (idx == 2) return vec3(1.0, 0.0, 0.0);   // East (+X)
	if (idx == 3) return vec3(-1.0, 0.0, 0.0);  // West (-X)
	if (idx == 4) return vec3(0.0, 1.0, 0.0);   // Top (+Y)
	if (idx == 5) return vec3(0.0, -1.0, 0.0);  // Bottom (-Y)
	return vec3(0.0, 0.0, 1.0); // Default
}

vec3 unpackRGB565(int val) {
	// R: 5 bits, G: 6 bits, B: 5 bits
	float r = float((val >> 11) & 0x1F) / 31.0;
	float g = float((val >> 5) & 0x3F) / 63.0;
	float b = float(val & 0x1F) / 31.0;
	return vec3(r, g, b);
}

void main() {
	vec3 pos = vec3(aPos);
	FragPos = pos;
	
	// Decode info
	// aData.x = Normal (low byte) | Brightness (high byte)
	// aData.y = TextureID
	// aData.z = Tint (RGB565)
	
	int info = int(aData.x);
	int normalIdx = info & 255;
	int brightnessVal = (info >> 8) & 255;
	
	int texID = int(aData.y);
	// Cast directly to int (handling signed/unsigned issue via bit logic if needed)
	// But since we use GL_UNSIGNED_SHORT in pointer, and "int" in shader, OpenGL converts float to int.
	// 65535.0 -> 65535.
	int tintVal = int(aData.z);

	Normal = decodeNormal(normalIdx);
	Brightness = float(brightnessVal) / 255.0;
	TintColor = unpackRGB565(tintVal);

	// Generate UVs based on world position and normal
	// pos is now in range [0,1] for X/Z and [0,1] for Y (bottom-left origin)
	vec2 uv = vec2(0.0);
	if (normalIdx == 0 || normalIdx == 1) { // North/South (Z) -> X, Y
		uv = vec2(pos.x, -pos.y); // Use -y for texture coordinates to orient correctly (top-down texture mapping)
	} else if (normalIdx == 2 || normalIdx == 3) { // East/West (X) -> Z, Y
		uv = vec2(pos.z, -pos.y);
	} else { // Top/Bottom (Y) -> X, Z
		uv = pos.xz;
	}
	
	TexCoord = vec3(uv, float(texID));

	gl_Position = proj * view * vec4(pos, 1.0);
}
