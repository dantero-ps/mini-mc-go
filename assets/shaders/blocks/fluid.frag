#version 330 core
out vec4 FragColor;

in vec3 FragPos;
in vec3 TexCoord;
in vec3 TintColor;

uniform sampler2DArray textureArray;
uniform vec3 cameraPos;
uniform int isUnderwater;

void main() {
    vec4 texColor = texture(textureArray, TexCoord);
    vec4 finalColor = texColor * vec4(TintColor, 1.0);

    float dist = length(FragPos - cameraPos);
    float fogFactor = 1.0 - exp(-dist * 0.15);
    fogFactor = clamp(fogFactor, 0.0, 1.0);
    vec3 waterFog = vec3(0.1, 0.3, 0.5);
    finalColor.rgb = mix(finalColor.rgb, waterFog, fogFactor);
    finalColor.a = mix(finalColor.a, 1.0, fogFactor * 0.7);

    if (finalColor.a < 0.1)
        discard;

    FragColor = finalColor;
}
