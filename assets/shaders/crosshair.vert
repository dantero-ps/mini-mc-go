#version 330 core
layout(location = 0) in vec2 aPos;
uniform float aspectRatio;
void main() {
    vec2 correctedPos = vec2(aPos.x / aspectRatio, aPos.y);
	gl_Position = vec4(correctedPos, 0.0, 1.0);
}