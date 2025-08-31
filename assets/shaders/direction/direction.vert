#version 330 core
layout(location = 0) in vec2 aPos;
uniform float aspectRatio;
uniform float rotation;    // Rotation angle in radians
uniform float positionX;   // X position offset
uniform float positionY;   // Y position offset
void main() {
    // Apply rotation
    float cosA = cos(rotation);
    float sinA = sin(rotation);
    vec2 rotatedPos = vec2(
        aPos.x * cosA - aPos.y * sinA,
        aPos.x * sinA + aPos.y * cosA
    );

    // Apply position offset and aspect ratio correction
    vec2 finalPos = vec2(
        (rotatedPos.x + positionX) / aspectRatio,
        rotatedPos.y + positionY
    );

    gl_Position = vec4(finalPos, 0.0, 1.0);
}
