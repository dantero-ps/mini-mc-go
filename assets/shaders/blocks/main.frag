#version 330 core
in vec3 Normal;
in vec3 FragPos;
in vec3 TexCoord; // u, v, layer
in float Brightness;
in vec3 TintColor;

uniform vec3 lightDir;
uniform sampler2DArray textureArray;
uniform vec3 cameraPos;
uniform int isUnderwater;
out vec4 FragColor;

void main() {
	vec4 texColor = texture(textureArray, TexCoord);
	if (texColor.a < 0.1) discard;
	texColor.rgb *= TintColor;
	vec3 col = texColor.rgb * Brightness;

	if (isUnderwater != 0) {
		float dist = length(FragPos - cameraPos);
		float fogFactor = 1.0 - exp(-dist * 0.08);
		fogFactor = clamp(fogFactor, 0.0, 1.0);
		vec3 waterFog = vec3(0.1, 0.3, 0.5);
		col = mix(col, waterFog, fogFactor);
	}

	FragColor = vec4(col, texColor.a);
}
