#version 330 core
in vec3 Normal;
in vec3 FragPos;
flat in int FaceIndex;
uniform vec3 lightDir;
out vec4 FragColor;

// Face colors as uniforms
uniform vec3 faceColorNorth;   // North
uniform vec3 faceColorSouth;   // South
uniform vec3 faceColorEast;    // East
uniform vec3 faceColorWest;    // West
uniform vec3 faceColorTop;     // Top
uniform vec3 faceColorBottom;  // Bottom
uniform vec3 faceColorDefault; // Default

void main() {
	vec3 n = normalize(Normal);
	float diff = max(dot(n, -lightDir), 0.5);
	float edge = pow(1.0 - max(dot(n, vec3(0,1,0)), 0.0), 3.0);

	// Get color based on face index
	vec3 faceColor;
	switch (FaceIndex) {
		case 0: faceColor = faceColorNorth; break;
		case 1: faceColor = faceColorSouth; break;
		case 2: faceColor = faceColorEast; break;
		case 3: faceColor = faceColorWest; break;
		case 4: faceColor = faceColorTop; break;
		case 5: faceColor = faceColorBottom; break;
		default: faceColor = faceColorDefault; break;
	}

	vec3 col = faceColor * diff * (1.0 - 0.1 * edge);
	FragColor = vec4(col, 1.0);
}
