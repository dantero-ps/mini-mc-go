#version 330 core
in vec2 TexCoord;

uniform sampler2D crosshairTexture;

out vec4 FragColor;

void main() {
    // Sample the texture directly
    // Minecraft uses the full texture color with special blend mode
    FragColor = texture(crosshairTexture, TexCoord);
}