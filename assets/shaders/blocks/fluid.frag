#version 330 core
out vec4 FragColor;

in vec3 FragPos;
in vec3 TexCoord;
in vec3 TintColor;

uniform sampler2DArray textureArray;

void main() {
    vec4 texColor = texture(textureArray, TexCoord);
    
    // Apply tint
    vec4 finalColor = texColor * vec4(TintColor, 1.0);
    
    if (finalColor.a < 0.1)
        discard;
        
    FragColor = finalColor;
}
