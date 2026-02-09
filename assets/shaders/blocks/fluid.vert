#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec2 aUV;
layout(location = 2) in float aTexID;
layout(location = 3) in vec3 aTint;

uniform mat4 view;
uniform mat4 proj;

out vec3 FragPos;
out vec3 TexCoord; // u, v, layer
out vec3 TintColor;

void main() {
    FragPos = aPos;
    TintColor = aTint;
    
    // Pass texture coordinates directly (u, v, layer)
    TexCoord = vec3(aUV, aTexID);

    gl_Position = proj * view * vec4(aPos, 1.0);
}
