package meshing

import (
	"math"
	"mini-mc/internal/world"
	"testing"
)

func init() {
	// BlockSolidTable normalde registry tarafından doldurulur.
	// Testlerde registry import etmekten kaçınmak için manuel olarak set ediyoruz.
	world.BlockSolidTable[world.BlockTypeStone] = true
	world.BlockSolidTable[world.BlockTypeDirt] = true
	world.BlockSolidTable[world.BlockTypeGrass] = true
}

// approxEqualF32 checks two float32 values are within epsilon of each other.
func approxEqualF32(a, b, epsilon float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= epsilon
}

// ---- computeFlowAngle tests ----

func TestComputeFlowAngle_FlowEast(t *testing.T) {
	w := world.NewEmpty()

	// Su (5,5,5) seviye 2
	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 2)

	// Doğu komşusu (6,5,5) seviye 5 → diff=+3, akış Doğu yönünde
	w.Set(6, 5, 5, world.BlockTypeWater)
	w.SetMeta(6, 5, 5, 5)

	// Diğer hava komşularının altını katı yap (kenar çekimi olmasın)
	w.Set(4, 4, 5, world.BlockTypeStone)
	w.Set(5, 4, 4, world.BlockTypeStone)
	w.Set(5, 4, 6, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Doğu yönünde akış beklendi, sentinel değer geldi: %f", angle)
	}
	// Doğu = +X. fdx=+, fdz=0 → atan2(0, positive) = 0
	if !approxEqualF32(angle, 0.0, 0.2) {
		t.Errorf("Açı ≈ 0 (Doğu) beklendi, got %f (%.1f°)", angle, float64(angle)*180/math.Pi)
	}
}

func TestComputeFlowAngle_FlowWest(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 1)

	// Batı komşusu (4,5,5) seviye 6 → diff=+5, akış Batı yönünde
	w.Set(4, 5, 5, world.BlockTypeWater)
	w.SetMeta(4, 5, 5, 6)

	w.Set(6, 4, 5, world.BlockTypeStone)
	w.Set(5, 4, 4, world.BlockTypeStone)
	w.Set(5, 4, 6, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Batı yönünde akış beklendi, sentinel geldi: %f", angle)
	}
	// Batı = -X. fdx=-, fdz=0 → atan2(0, negative) = ±π
	absAngle := float64(angle)
	if absAngle < 0 {
		absAngle = -absAngle
	}
	if math.Abs(absAngle-math.Pi) > 0.2 {
		t.Errorf("Açı ≈ ±π (Batı) beklendi, got %f (%.1f°)", angle, float64(angle)*180/math.Pi)
	}
}

func TestComputeFlowAngle_FlowSouth(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 1)

	// Güney komşusu (5,5,6) seviye 4 → diff=+3
	w.Set(5, 5, 6, world.BlockTypeWater)
	w.SetMeta(5, 5, 6, 4)

	w.Set(4, 4, 5, world.BlockTypeStone)
	w.Set(6, 4, 5, world.BlockTypeStone)
	w.Set(5, 4, 4, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Güney yönünde akış beklendi, sentinel geldi: %f", angle)
	}
	// Güney = +Z. fdx=0, fdz=+ → atan2(positive, 0) = π/2
	expected := float32(math.Pi / 2)
	if !approxEqualF32(angle, expected, 0.2) {
		t.Errorf("Açı ≈ π/2 (Güney) beklendi, got %f (%.1f°)", angle, float64(angle)*180/math.Pi)
	}
}

func TestComputeFlowAngle_FlowNorth(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 2)

	// Kuzey komşusu (5,5,4) seviye 7 → diff=+5
	w.Set(5, 5, 4, world.BlockTypeWater)
	w.SetMeta(5, 5, 4, 7)

	w.Set(4, 4, 5, world.BlockTypeStone)
	w.Set(6, 4, 5, world.BlockTypeStone)
	w.Set(5, 4, 6, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Kuzey yönünde akış beklendi, sentinel geldi: %f", angle)
	}
	// Kuzey = -Z. atan2(negative, 0) = -π/2 → normalize → 3π/2 ≈ 4.712
	expected := float32(3 * math.Pi / 2)
	if !approxEqualF32(angle, expected, 0.2) {
		t.Errorf("Açı ≈ 3π/2 (Kuzey) beklendi, got %f (%.1f°)", angle, float64(angle)*180/math.Pi)
	}
}

func TestComputeFlowAngle_NoFlow_SameLevelNeighbors(t *testing.T) {
	w := world.NewEmpty()

	// Merkez blok
	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 3)

	// 4 komşu aynı seviyede → diff=0, akış yok
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		w.Set(5+d[0], 5, 5+d[1], world.BlockTypeWater)
		w.SetMeta(5+d[0], 5, 5+d[1], 3)
	}

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle != -1.0 {
		t.Errorf("Tüm komşular aynı seviyede: -1.0 (akış yok) beklendi, got %f", angle)
	}
}

func TestComputeFlowAngle_NoFlow_AllSolid(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 2)

	// Tüm yatay komşular katı
	w.Set(6, 5, 5, world.BlockTypeStone)
	w.Set(4, 5, 5, world.BlockTypeStone)
	w.Set(5, 5, 6, world.BlockTypeStone)
	w.Set(5, 5, 4, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle != -1.0 {
		t.Errorf("Katı komşular: -1.0 (akış yok) beklendi, got %f", angle)
	}
}

func TestComputeFlowAngle_DropOffEast(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 2)

	// Doğu (6,5,5) hava, altı (6,4,5) da hava → kenar çekimi
	// Diğer komşular katı (yalıtım)
	w.Set(4, 5, 5, world.BlockTypeStone)
	w.Set(5, 5, 6, world.BlockTypeStone)
	w.Set(5, 5, 4, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Kenar çekimiyle Doğu akışı beklendi, sentinel geldi: %f", angle)
	}
	// fdx=8.0, fdz=0 → atan2(0, 8) = 0
	if !approxEqualF32(angle, 0.0, 0.2) {
		t.Errorf("Açı ≈ 0 (Doğu, kenar çekimi) beklendi, got %f (%.1f°)", angle, float64(angle)*180/math.Pi)
	}
}

func TestComputeFlowAngle_DropOffBlockedByFloor(t *testing.T) {
	w := world.NewEmpty()

	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 2)

	// Doğu hava ama altı katı (düşüş yok) → kenar çekimi tetiklenmez
	w.Set(6, 4, 5, world.BlockTypeStone) // Doğu altı katı
	w.Set(4, 5, 5, world.BlockTypeStone)
	w.Set(5, 5, 6, world.BlockTypeStone)
	w.Set(5, 5, 4, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	// Katı zemin üstündeki hava kenar çekimi yaratmaz → akış yok
	if angle != -1.0 {
		t.Errorf("Zemin katı olduğunda kenar çekimi olmamalı: -1.0 beklendi, got %f", angle)
	}
}

func TestComputeFlowAngle_FallingWater_TreatedAsSource(t *testing.T) {
	w := world.NewEmpty()

	// Düşen su (meta=8) → efektif seviye 0 (kaynak gibi davranır)
	w.Set(5, 5, 5, world.BlockTypeWater)
	w.SetMeta(5, 5, 5, 8)

	// Doğu komşusu seviye 5 → diff = 5-0 = 5
	w.Set(6, 5, 5, world.BlockTypeWater)
	w.SetMeta(6, 5, 5, 5)

	w.Set(4, 5, 5, world.BlockTypeStone)
	w.Set(5, 5, 6, world.BlockTypeStone)
	w.Set(5, 5, 4, world.BlockTypeStone)

	angle := computeFlowAngle(w, 5, 5, 5, world.BlockTypeWater)

	if angle < -0.5 {
		t.Fatalf("Düşen su için yönlü akış beklendi, sentinel geldi: %f", angle)
	}
	if !approxEqualF32(angle, 0.0, 0.2) {
		t.Errorf("Düşen su: açı ≈ 0 (Doğu) beklendi, got %f", angle)
	}
}

// ---- getFluidHeight tests ----

func TestGetFluidHeight_FluidAbove_ReturnsOne(t *testing.T) {
	w := world.NewEmpty()

	// (5,5,5) kaynak su, üstünde (5,6,5) de kaynak su
	w.Set(5, 5, 5, world.BlockTypeWater)
	w.Set(5, 6, 5, world.BlockTypeWater)

	// getFluidHeight(w, 5, 5, 5) → corner (5,5,5) için 4 köşe bloğu kontrol eder.
	// Köşelerden biri (5,5,5)'in üstü su → erken dönüş 1.0
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	if h != 1.0 {
		t.Errorf("Üstte aynı su varken 1.0 beklendi, got %f", h)
	}
}

func TestGetFluidHeight_AllAir_ReturnsZero(t *testing.T) {
	w := world.NewEmpty()
	// Su bloğu yok, hava dolu dünya

	// Hava köşeler: sum += 1.0 her biri için, count++
	// result = 1.0 - (4/4) = 0.0
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	if h != 0.0 {
		t.Errorf("Tüm köşeler hava: 0.0 beklendi, got %f", h)
	}
}

func TestGetFluidHeight_AllSolid_ReturnsZero(t *testing.T) {
	w := world.NewEmpty()

	// Tüm 4 köşe bloğu katı (contribution = 0, count = 0)
	w.Set(5, 5, 5, world.BlockTypeStone)
	w.Set(4, 5, 5, world.BlockTypeStone)
	w.Set(5, 5, 4, world.BlockTypeStone)
	w.Set(4, 5, 4, world.BlockTypeStone)

	// Üstte su yok
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	if h != 0.0 {
		t.Errorf("Tüm köşeler katı: 0.0 beklendi, got %f", h)
	}
}

func TestGetFluidHeight_SourceWater_CorrectHeight(t *testing.T) {
	w := world.NewEmpty()

	// 4 köşe bloğu da kaynak su (meta=0)
	w.Set(5, 5, 5, world.BlockTypeWater) // dx=0, dz=0
	w.Set(4, 5, 5, world.BlockTypeWater) // dx=-1, dz=0
	w.Set(5, 5, 4, world.BlockTypeWater) // dx=0, dz=-1
	w.Set(4, 5, 4, world.BlockTypeWater) // dx=-1, dz=-1
	// meta=0 varsayılan (SetMeta çağrılmadı)

	// Üstte su yok (yükseklik hesabı yapılır)
	// meta=0 → GetLiquidHeightPercent(0) = 1/9
	// Her köşe: sum += 10/9 + 1/9 = 11/9, count += 11
	// 4 köşe: sum = 44/9, count = 44
	// result = 1.0 - (44/9)/44 = 1.0 - 1/9 ≈ 0.8889
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	expected := float32(1.0 - 1.0/9.0)
	if !approxEqualF32(h, expected, 0.001) {
		t.Errorf("Kaynak su yüksekliği ≈ %.4f beklendi, got %.4f", expected, h)
	}
}

func TestGetFluidHeight_Level3Water_CorrectHeight(t *testing.T) {
	w := world.NewEmpty()

	// 4 köşe bloğu seviye 3 akan su
	for _, pos := range [][3]int{{5, 5, 5}, {4, 5, 5}, {5, 5, 4}, {4, 5, 4}} {
		w.Set(pos[0], pos[1], pos[2], world.BlockTypeWater)
		w.SetMeta(pos[0], pos[1], pos[2], 3)
	}

	// meta=3: not source, not falling → ×10 ağırlık yok
	// GetLiquidHeightPercent(3) = (3+1)/9 = 4/9
	// Her köşe: sum += 4/9, count += 1
	// 4 köşe: sum = 16/9, count = 4
	// result = 1.0 - (16/9)/4 = 1.0 - 4/9 ≈ 0.5556
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	expected := float32(1.0 - 4.0/9.0)
	if !approxEqualF32(h, expected, 0.001) {
		t.Errorf("Seviye-3 su yüksekliği ≈ %.4f beklendi, got %.4f", expected, h)
	}
}

func TestGetFluidHeight_MixedCorners(t *testing.T) {
	w := world.NewEmpty()

	// 2 köşe su (meta=0 kaynak), 2 köşe hava
	w.Set(5, 5, 5, world.BlockTypeWater) // dx=0, dz=0: kaynak
	w.Set(4, 5, 5, world.BlockTypeWater) // dx=-1, dz=0: kaynak
	// (5,5,4) ve (4,5,4) hava

	// Kaynak köşe: sum += 11/9, count += 11
	// Hava köşe: sum += 1.0 = 9/9, count += 1
	// Toplam: sum = 2*(11/9) + 2*(9/9) = 22/9 + 18/9 = 40/9, count = 2*11 + 2*1 = 24
	// result = 1.0 - (40/9)/24 = 1.0 - 40/216 ≈ 0.815
	h := getFluidHeight(w, 5, 5, 5, world.BlockTypeWater)

	expected := float32(1.0 - 40.0/216.0)
	if !approxEqualF32(h, expected, 0.001) {
		t.Errorf("Karışık köşeler: %.4f beklendi, got %.4f", expected, h)
	}
}
