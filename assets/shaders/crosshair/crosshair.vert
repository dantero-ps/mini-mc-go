#version 330 core
layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aTexCoord;

uniform int screenWidth;
uniform int screenHeight;

out vec2 TexCoord;

void main() {
    // Convert pixel coordinates to normalized device coordinates (-1 to 1)
    // Center the crosshair at screen center
    float x = (aPos.x + float(screenWidth) / 2.0) / float(screenWidth) * 2.0 - 1.0;
    float y = (aPos.y + float(screenHeight) / 2.0) / float(screenHeight) * 2.0 - 1.0;
    
    gl_Position = vec4(x, y, 0.0, 1.0);
    
    // Pass texture coordinates to fragment shader
    // Normalize to 0-1 range (widgets.png is 256x256, crosshair is at 0,0 with 16x16 size)
    TexCoord = aTexCoord / 256.0;
}