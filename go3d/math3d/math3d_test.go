package math3d

import (
	"math"
	"testing"
)

func close(a, b, eps float32) bool { return float32(math.Abs(float64(a-b))) <= eps }

func TestMulIdentity(t *testing.T) {
	m := RotateY(0.7)
	r := Mul(m, Identity())
	for i := range m {
		if !close(m[i], r[i], 1e-5) {
			t.Fatalf("A*I != A at %d: %v vs %v", i, m[i], r[i])
		}
	}
}

func TestMulOrder(t *testing.T) {
	// T(1,0,0) * Rz(90°) 作用在 (1,0,0)：先旋转成 (0,1,0)，再平移到 (1,1,0)
	m := Mul(Translate(1, 0, 0), RotateZ(math.Pi/2))
	v := m.Transform(Vec3{1, 0, 0})
	if !close(v.X, 1, 1e-4) || !close(v.Y, 1, 1e-4) || !close(v.Z, 0, 1e-4) {
		t.Fatalf("T*Rz order wrong: %v", v)
	}
}

func TestRotateOrthogonal(t *testing.T) {
	m := Mul(Mul(RotateX(0.3), RotateY(0.5)), RotateZ(0.8))
	v := m.Transform(Vec3{1, 0, 0})
	if !close(v.Length(), 1, 1e-4) {
		t.Fatalf("rotation not length-preserving: %v", v)
	}
}

func TestPerspectiveNearPlane(t *testing.T) {
	p := Perspective(math.Pi/3, 16.0/9.0, 0.1, 100)
	// 相机前方 near 处，中心点 → NDC (0,0,-1)
	x, y, z, w := p.TransformW(Vec3{0, 0, -0.1})
	if w <= 0 {
		t.Fatalf("w must be positive, got %v", w)
	}
	if !close(x/w, 0, 1e-4) || !close(y/w, 0, 1e-4) || !close(z/w, -1, 1e-4) {
		t.Fatalf("near plane mapping wrong: %v %v %v", x/w, y/w, z/w)
	}
}

func TestPerspectiveFarPlane(t *testing.T) {
	p := Perspective(math.Pi/3, 16.0/9.0, 0.1, 100)
	_, _, z, w := p.TransformW(Vec3{0, 0, -100})
	if !close(z/w, 1, 1e-4) {
		t.Fatalf("far plane z/w should be 1, got %v", z/w)
	}
}

func TestLookAtEyeAtOrigin(t *testing.T) {
	// 相机在原点看向 -Z，up=+Y → 视图矩阵恒等
	m := LookAt(Vec3{0, 0, 0}, Vec3{0, 0, -1}, Vec3{0, 1, 0})
	for i := range m {
		want := float32(0)
		if i%5 == 0 {
			want = 1
		}
		if !close(m[i], want, 1e-5) {
			t.Fatalf("LookAt origin identity wrong at %d: %v", i, m[i])
		}
	}
}

func TestLookAtMovesWorldBehindCamera(t *testing.T) {
	// 相机在 (0,0,5) 看向原点：原点应变换到视图空间 (0,0,-5)
	m := LookAt(Vec3{0, 0, 5}, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	v := m.Transform(Vec3{0, 0, 0})
	if !close(v.X, 0, 1e-4) || !close(v.Y, 0, 1e-4) || !close(v.Z, -5, 1e-3) {
		t.Fatalf("LookAt translation wrong: %v", v)
	}
}

func TestFPSViewForward(t *testing.T) {
	// yaw=0 pitch=0 → 前向 -Z
	f := Forward(0, 0)
	if !close(f.Z, -1, 1e-4) || !close(f.X, 0, 1e-4) || !close(f.Y, 0, 1e-4) {
		t.Fatalf("Forward(0,0) should be -Z: %v", f)
	}
	// yaw=π/2 → 前向 +X
	f = Forward(math.Pi/2, 0)
	if !close(f.X, 1, 1e-4) || !close(f.Z, 0, 1e-4) {
		t.Fatalf("Forward(pi/2) should be +X: %v", f)
	}
}

func TestInvertRoundTrip(t *testing.T) {
	// 构造一个复合变换（平移+旋转+缩放），M * M⁻¹ ≈ I
	m := Mul(Mul(Mul(Translate(1, 2, 3), RotateY(0.7)), RotateX(-0.4)), Scale(2))
	inv, ok := m.Invert()
	if !ok {
		t.Fatal("matrix not invertible")
	}
	id := Mul(m, inv)
	for i := range id {
		want := float32(0)
		if i%5 == 0 {
			want = 1
		}
		if !close(id[i], want, 1e-3) {
			t.Fatalf("M*M⁻¹ != I at %d: %v", i, id[i])
		}
	}
}

func TestUnproject(t *testing.T) {
	// 投影 + 逆投影：世界点 → clip → ndc → 反投影应还原
	proj := Perspective(math.Pi/3, 16.0/9.0, 0.1, 100)
	view := LookAt(Vec3{0, 2, 5}, Vec3{0, 0, 0}, Vec3{0, 1, 0})
	vp := Mul(proj, view)
	p := Vec3{0.5, -0.3, 1.2}
	x, y, z, w := vp.TransformW(p)
	back, ok := vp.Unproject(x/w, y/w, z/w)
	if !ok {
		t.Fatal("unproject failed")
	}
	if !close(back.X, p.X, 1e-3) || !close(back.Y, p.Y, 1e-3) || !close(back.Z, p.Z, 1e-3) {
		t.Fatalf("unproject mismatch: %v vs %v", back, p)
	}
}
