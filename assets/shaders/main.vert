#version 330 core
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