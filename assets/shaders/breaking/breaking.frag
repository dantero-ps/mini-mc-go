#version 410 core
out vec4 FragColor;

in vec2 TexCoord;

uniform sampler2DArray breakingTexture;
uniform int layer;

void main() {
    vec4 texColor = texture(breakingTexture, vec3(TexCoord, float(layer)));
    
    // Minecraft uses GL_DST_COLOR, GL_SRC_COLOR blending (Result = 2 * Src * Dst)
    // To leave the block unchanged (neutral), we need Src = 0.5 (since 2 * 0.5 * Dst = Dst)
    // We mix between 0.5 (transparent areas) and the texture color (cracks)
    vec3 color = mix(vec3(0.5), texColor.rgb, texColor.a);
    
    FragColor = vec4(color, 1.0);
}
