package world

import (
	"math/rand"
)

// ChunkProvider189 implements the Minecraft 1.8.9 chunk generation logic.
type ChunkProvider189 struct {
	rnd *rand.Rand

	minLimitNoise *AuthenticNoiseGeneratorOctaves
	maxLimitNoise *AuthenticNoiseGeneratorOctaves
	mainNoise     *AuthenticNoiseGeneratorOctaves
	surfaceNoise  *AuthenticNoiseGeneratorOctaves
	scaleNoise    *AuthenticNoiseGeneratorOctaves
	depthNoise    *AuthenticNoiseGeneratorOctaves
	forestNoise   *AuthenticNoiseGeneratorOctaves
}

func NewChunkProvider189(seed int64) *ChunkProvider189 {
	rnd := rand.New(rand.NewSource(seed))

	return &ChunkProvider189{
		rnd:           rnd,
		minLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		maxLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		mainNoise:     NewAuthenticNoiseGeneratorOctaves(rnd, 8),
		surfaceNoise:  NewAuthenticNoiseGeneratorOctaves(rnd, 4),
		scaleNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 10),
		depthNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		forestNoise:   NewAuthenticNoiseGeneratorOctaves(rnd, 8),
	}
}
