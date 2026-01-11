#version 330 core
in vec3 Normal;
in vec3 FragPos;
in vec3 TexCoord; // u, v, layer
in float Brightness;
in vec3 TintColor;

uniform vec3 lightDir;
uniform sampler2DArray textureArray;
out vec4 FragColor;

void main() {
	// Sample texture from array
	vec4 texColor = texture(textureArray, TexCoord);

	// Apply Tint Color (Passed from CPU)
	texColor.rgb *= TintColor;

	// Apply brightness (Calculated on CPU)
	vec3 col = texColor.rgb * Brightness;
	
	FragColor = vec4(col, texColor.a);
}
