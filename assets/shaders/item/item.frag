#version 330 core
out vec4 FragColor;

in vec2 TexCoord;
in vec3 Normal;

uniform sampler2DArray textureArray;
uniform int textureID;
uniform vec3 tintColor;

void main() {
    // Basic diffuse lighting similar to blocks
    vec3 lightDir = normalize(vec3(0.3, 1.0, 0.3));
    // Ambient 0.6, Diffuse 0.4
    float diff = max(dot(normalize(Normal), lightDir), 0.6);

    vec4 texColor = texture(textureArray, vec3(TexCoord, float(textureID)));
    if(texColor.a < 0.1) discard;

    // Apply tint and lighting
    vec3 finalColor = texColor.rgb * tintColor * diff;
    FragColor = vec4(finalColor, texColor.a);
}
