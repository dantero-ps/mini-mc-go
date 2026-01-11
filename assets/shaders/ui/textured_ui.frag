#version 410 core
out vec4 FragColor;

in vec2 TexCoord;

uniform sampler2D uTexture;
uniform vec4 uColor;

void main() {
    vec4 texColor = texture(uTexture, TexCoord);
    if(texColor.a < 0.1) discard;
    FragColor = texColor * uColor;
}

