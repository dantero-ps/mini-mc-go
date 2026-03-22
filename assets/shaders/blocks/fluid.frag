#version 330 core
out vec4 FragColor;

in vec3 FragPos;
in vec3 TexCoord;
in vec3 TintColor;
in float FlowAngle;

uniform sampler2DArray textureArray;
uniform vec3 cameraPos;
uniform int isUnderwater;
uniform float time;

void main() {
    vec2 animUV = TexCoord.xy;

    if (FlowAngle < -2.5) {
        // Bottom face: no animation
    } else if (FlowAngle < -1.5) {
        // Side face: scroll downward (subtract = features move down)
        animUV.y -= time * 0.4;
    } else if (FlowAngle < -0.5) {
        // Still water top: slow isotropic scroll
        animUV.x -= time * 0.03;
        animUV.y -= time * 0.03;
    } else {
        // Directional flowing water top: subtract flowDir so texture moves WITH the flow
        vec2 flowDir = vec2(cos(FlowAngle), sin(FlowAngle));
        animUV -= flowDir * time * 0.15;
    }

    // Rotate UV 90° CCW: (u, v) -> (-v, u)
    vec2 sampUV = vec2(-animUV.y, animUV.x);
    vec4 texColor = texture(textureArray, vec3(sampUV, TexCoord.z));
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
