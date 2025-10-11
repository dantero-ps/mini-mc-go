#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in float aEncodedNormal;
layout(location = 2) in vec3 instancePos;
uniform mat4 view;
uniform mat4 proj;
out vec3 Normal;
out vec3 FragPos;
flat out int FaceIndex;

// Decode normal from encoded value
vec3 decodeNormal(float encoded) {
	int idx = int(encoded + 0.5);
	if (idx == 0) return vec3(0.0, 0.0, 1.0);   // North (+Z)
	if (idx == 1) return vec3(0.0, 0.0, -1.0);  // South (-Z)
	if (idx == 2) return vec3(1.0, 0.0, 0.0);   // East (+X)
	if (idx == 3) return vec3(-1.0, 0.0, 0.0);  // West (-X)
	if (idx == 4) return vec3(0.0, 1.0, 0.0);   // Top (+Y)
	if (idx == 5) return vec3(0.0, -1.0, 0.0);  // Bottom (-Y)
	return vec3(0.0, 0.0, 1.0); // Default
}

void main() {
	vec3 pos = aPos + instancePos;
	FragPos = pos;
	
	// Decode normal and set FaceIndex
	FaceIndex = int(aEncodedNormal + 0.5);
	Normal = decodeNormal(aEncodedNormal);

	gl_Position = proj * view * vec4(pos, 1.0);
}
