#version 330 core
layout(location = 0) in vec3 aPos;
layout(location = 1) in vec2 aUV;
layout(location = 2) in float aTexID;
layout(location = 3) in vec3 aTint;
layout(location = 4) in float aFlowAngle;

uniform mat4 view;
uniform mat4 proj;

out vec3 FragPos;
out vec3 TexCoord; // u, v, layer
out vec3 TintColor;
out float FlowAngle;

void main() {
    FragPos = aPos;
    TintColor = aTint;
    FlowAngle = aFlowAngle;
    TexCoord = vec3(aUV, aTexID);
    gl_Position = proj * view * vec4(aPos, 1.0);
}
