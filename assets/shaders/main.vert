#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
layout(location = 2) in vec3 instancePos;
uniform mat4 view;
uniform mat4 proj;
out vec3 Normal;
out vec3 FragPos;
flat out int FaceIndex;
void main() {
	vec3 pos = aPos + instancePos;
	FragPos = pos;
	Normal = aNormal;

	// Determine face index based on normal
	if (aNormal.z > 0.5) FaceIndex = 0;      // North
	else if (aNormal.z < -0.5) FaceIndex = 1; // South
	else if (aNormal.x > 0.5) FaceIndex = 2;  // East
	else if (aNormal.x < -0.5) FaceIndex = 3; // West
	else if (aNormal.y > 0.5) FaceIndex = 4;  // Top
	else if (aNormal.y < -0.5) FaceIndex = 5; // Bottom
	else FaceIndex = 6;                       // Default

	gl_Position = proj * view * vec4(pos, 1.0);
}
