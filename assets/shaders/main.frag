#version 330 core
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