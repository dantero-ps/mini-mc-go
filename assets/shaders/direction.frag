#version 330 core
out vec4 FragColor;
uniform vec3 directionColor;
void main() {
    FragColor = vec4(directionColor, 0.8);
}