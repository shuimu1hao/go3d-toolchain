package render

import (
	"testing"

	"go3d/math3d"
	"go3d/mesh"
)

// makePixels 构造离屏像素缓冲（B,G,R,pad）。
func makePixels(w, h int) []byte {
	return make([]byte, w*h*4)
}

// countNonBg 统计非背景像素数。背景由 Clear 固定为 B30/G22/R18。
func countNonBg(pix []byte) int {
	n := 0
	for i := 0; i < len(pix); i += 4 {
		if pix[i] != 30 || pix[i+1] != 22 || pix[i+2] != 18 {
			n++
		}
	}
	return n
}

func TestRenderCubeVisible(t *testing.T) {
	w, h := 320, 200
	pix := makePixels(w, h)
	rd := NewRenderer(w, h)
	cam := DefaultCamera()
	cam.Pos = math3d.Vec3{0, 1.5, 4} // +Z 侧看向 -Z，立方体在原点前方
	cam.Yaw = 0
	objs := []Object{{Mesh: mesh.Cube(1.4), Pos: math3d.Vec3{0, 0.8, 0}, Scale: 1}}

	rd.Clear(pix)
	tris := rd.Render(pix, cam, objs)
	if tris == 0 {
		t.Fatal("no triangles rendered")
	}
	n := countNonBg(pix)
	// 立方体应占据可见像素：至少 500 像素（320x200 下约占 ~5%+）
	if n < 500 {
		t.Fatalf("cube too small: %d non-bg pixels (%d tris)", n, tris)
	}
}

func TestRenderDepthOcclusion(t *testing.T) {
	// 大球在前挡住小球：两个物体同一位置时，前面的必须完全遮挡后面的。
	w, h := 320, 200
	pix := makePixels(w, h)
	rd := NewRenderer(w, h)
	cam := DefaultCamera()
	cam.Pos = math3d.Vec3{0, 0, 6}
	cam.Yaw = 0
	cam.Pitch = 0

	// 单看大球
	big := Object{Mesh: mesh.Sphere(1.8, 20, 10), Pos: math3d.Vec3{0, 0, 0}, Scale: 1}
	rd.Clear(pix)
	rd.Render(pix, cam, []Object{big})
	bigPixels := countNonBg(pix)

	// 大球 + 相同位置的更小球（球半径不同：在内部再放一个小球，应完全被遮挡）
	small := Object{Mesh: mesh.Sphere(1.0, 16, 8), Pos: math3d.Vec3{0, 0, 0}, Scale: 1}
	rd.Clear(pix)
	rd.Render(pix, cam, []Object{big, small})
	bothPixels := countNonBg(pix)

	// z-buffer 正常工作：小球完全被大球遮挡，像素数不变
	if bothPixels != bigPixels {
		t.Fatalf("occlusion broken: big=%d big+small=%d", bigPixels, bothPixels)
	}
}

func TestBackfaceCulling(t *testing.T) {
	// 单个三角形：从正面看到（法线朝向相机），背面看不到。
	w, h := 320, 200
	pix := makePixels(w, h)
	rd := NewRenderer(w, h)
	cam := DefaultCamera()
	cam.Pos = math3d.Vec3{0, 0, 4}
	cam.Yaw = 0
	cam.Pitch = 0

	// 一个面向 +Z 的三角形（法线 +Z，朝向相机）
	m := &mesh.Mesh{
		Positions: []math3d.Vec3{
			{-1, -1, 0}, {1, -1, 0}, {0, 1, 0},
		},
		Faces: []mesh.Face{{A: 0, B: 1, C: 2, Col: mesh.Col(200, 100, 50)}},
	}
	// 验证法线方向：顶点 0→1→2 逆时针（从 +Z 看）
	rd.Clear(pix)
	tris := rd.Render(pix, cam, []Object{{Mesh: m}})
	if tris != 1 {
		t.Fatalf("front-facing tri should render, got tris=%d", tris)
	}
	if n := countNonBg(pix); n < 100 {
		t.Fatalf("front-facing tri too few pixels: %d", n)
	}

	// 翻转环绕顺序 → 背面 → 应被剔除
	m2 := &mesh.Mesh{
		Positions: []math3d.Vec3{
			{-1, -1, 0}, {1, -1, 0}, {0, 1, 0},
		},
		Faces: []mesh.Face{{A: 0, B: 2, C: 1, Col: mesh.Col(200, 100, 50)}},
	}
	rd.Clear(pix)
	tris = rd.Render(pix, cam, []Object{{Mesh: m2}})
	if tris != 0 {
		t.Fatalf("back-facing tri should be culled, got tris=%d", tris)
	}
}

func TestNearClip(t *testing.T) {
	// 三角形跨过相机近平面：部分在相机后面，应被裁剪而不是画错。
	w, h := 320, 200
	pix := makePixels(w, h)
	rd := NewRenderer(w, h)
	cam := DefaultCamera()
	cam.Pos = math3d.Vec3{0, 0, 3} // +Z 侧看向 -Z
	cam.Yaw = 0
	cam.Pitch = 0

	// 三角形：两个顶点在近平面之后（世界 z=3.5 > 相机 z=3 的近裁剪边界），
	// 一个顶点在相机前方深处（z=-2）。法线朝 +Z 偏上，面向相机。
	m := &mesh.Mesh{
		Positions: []math3d.Vec3{
			{-1, -1, 3.5}, {1, -1, 3.5}, {0, 1, -2},
		},
		Faces: []mesh.Face{{A: 0, B: 1, C: 2, Col: mesh.Col(50, 200, 100)}},
	}
	rd.Clear(pix)
	tris := rd.Render(pix, cam, []Object{{Mesh: m}})
	if tris == 0 {
		t.Fatal("crossing triangle should still render after near clip")
	}
	if n := countNonBg(pix); n < 10 {
		t.Fatalf("crossing triangle rendered too few pixels: %d", n)
	}
}

func TestWireframeMode(t *testing.T) {
	w, h := 320, 200
	pix := makePixels(w, h)
	rd := NewRenderer(w, h)
	cam := DefaultCamera()
	cam.Pos = math3d.Vec3{0, 1.5, 4}
	cam.Yaw = 0
	objs := []Object{{Mesh: mesh.Cube(1.4), Pos: math3d.Vec3{0, 0.8, 0}, Scale: 1}}

	rd.Clear(pix)
	rd.Render(pix, cam, objs)
	filled := countNonBg(pix)

	rd.Wireframe = true
	rd.Clear(pix)
	rd.Render(pix, cam, objs)
	wire := countNonBg(pix)

	// 线框应明显少于实体，但不是 0
	if wire >= filled {
		t.Fatalf("wireframe should have fewer pixels: filled=%d wire=%d", filled, wire)
	}
	if wire < 50 {
		t.Fatalf("wireframe too sparse: %d", wire)
	}
}
