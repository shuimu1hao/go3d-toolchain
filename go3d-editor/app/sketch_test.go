package app

import (
	"os"
	"path/filepath"
	"testing"

	"go3d/math3d"
	"go3d/mesh"
)

// ---------- 草图 ----------

func TestTriangulateEarSquare(t *testing.T) {
	pts := RectSketch(0, 0, 1, 1)
	tris := TriangulateEar(pts)
	if len(tris) != 2 {
		t.Fatalf("square should give 2 tris, got %d", len(tris))
	}
	// 面积守恒
	area := float32(0)
	for _, tr := range tris {
		area += triArea2(pts[tr[0]], pts[tr[1]], pts[tr[2]]) / 2
	}
	if area < 0.99 || area > 1.01 {
		t.Fatalf("area should be 1, got %v", area)
	}
}

func TestTriangulateEarConcave(t *testing.T) {
	// 凹多边形（L 形）
	pts := []Vec2{
		{0, 0}, {2, 0}, {2, 1}, {1, 1}, {1, 2}, {0, 2},
	}
	tris := TriangulateEar(pts)
	if len(tris) == 0 {
		t.Fatal("concave polygon should triangulate")
	}
	if len(tris) != 4 {
		t.Fatalf("L-shape should give 4 tris, got %d", len(tris))
	}
}

func TestExtrudeMeshVolume(t *testing.T) {
	pts := RectSketch(0, 0, 2, 2)
	m := ExtrudeMesh(pts, PlaneXY, 1, mesh.Col(100, 100, 100))
	if m == nil {
		t.Fatal("extrude failed")
	}
	// 顶点数：底 4 + 顶 4 = 8
	if len(m.Positions) != 8 {
		t.Fatalf("expect 8 verts, got %d", len(m.Positions))
	}
	// 面数：底 2 + 顶 2 + 侧 4*2 = 12
	if len(m.Faces) != 12 {
		t.Fatalf("expect 12 faces, got %d", len(m.Faces))
	}
	// 用三角面体积法验证体积 ≈ 4（2x2x1）
	vol := meshVolume(m)
	if vol < 3.5 || vol > 4.5 {
		t.Fatalf("volume should be ~4, got %v", vol)
	}
}

func meshVolume(m *mesh.Mesh) float32 {
	// 有符号体积：Σ signed volume of tetrahedron(origin, tri)
	vol := float32(0)
	for i := range m.Faces {
		f := &m.Faces[i]
		a := m.Positions[f.A]
		b := m.Positions[f.B]
		c := m.Positions[f.C]
		vol += a.Dot(b.Cross(c)) / 6
	}
	if vol < 0 {
		vol = -vol
	}
	return vol
}

// ---------- CSG ----------

// raycastFirst 从 p 沿 dir 找最近命中面的距离（返回命中距离，-1 未命中）。
func raycastFirst(m *mesh.Mesh, p, dir math3d.Vec3) float32 {
	best := float32(-1)
	for i := range m.Faces {
		f := &m.Faces[i]
		a := m.Positions[f.A]
		b := m.Positions[f.B]
		c := m.Positions[f.C]
		n := b.Sub(a).Cross(c.Sub(a))
		dn := n.Dot(dir)
		if dn > -1e-8 && dn < 1e-8 {
			continue
		}
		t := a.Sub(p).Dot(n) / dn
		if t <= 1e-4 {
			continue
		}
		q := p.Add(dir.MulScalar(t))
		if pointInTri3D(q, a, b, c) {
			if best < 0 || t < best {
				best = t
			}
		}
	}
	return best
}

func TestCSGSubtractHole(t *testing.T) {
	// 大立方体 - 小立方体（部分重叠）→ 挖出孔洞，洞内有内表面
	big := mesh.Cube(2.0) // -1..1，体积 8
	small := translateMesh(mesh.Cube(1.0), math3d.Vec3{0.3, 0, 0}) // -0.2..0.8
	res := CSGBoolean(big, small, CSGSubtract, mesh.Col(100, 100, 100), mesh.Col(200, 200, 200))
	if res == nil || len(res.Faces) == 0 {
		t.Fatal("CSG subtract produced empty result")
	}
	// 面数应显著多于原 A（12 面）：含被切的内表面（bbox 优化后恰好 12+12=24）
	if len(res.Faces) < 20 {
		t.Fatalf("subtract should create inner surfaces, got %d", len(res.Faces))
	}
	// 洞内点 (0.3,0,0) 沿 +X 应有内表面（B 的 x=0.8 面）在 ~0.5 距离处
	d := raycastFirst(res, math3d.Vec3{0.3, 0, 0}, math3d.Vec3{1, 0, 0})
	if d < 0 || d > 0.7 {
		t.Fatalf("hole inner wall should be near x=0.8 (dist ~0.5), got %v", d)
	}
	// 未挖空区域：(-0.5,0,0) 沿 +X 第一个面在 x=-0.2 处（洞左壁）距离 0.3
	d2 := raycastFirst(res, math3d.Vec3{-0.5, 0, 0}, math3d.Vec3{1, 0, 0})
	if d2 < 0 || d2 > 0.45 {
		t.Fatalf("hole left wall should be near x=-0.2 (dist ~0.3), got %v", d2)
	}
}

func TestCSGUnionSurface(t *testing.T) {
	// 两个错开的立方体（重叠 0.3）：并集表面应连贯
	a := translateMesh(mesh.Cube(1.0), math3d.Vec3{-0.4, 0, 0}) // -0.9..0.1
	b := translateMesh(mesh.Cube(1.0), math3d.Vec3{0.3, 0, 0})  // -0.2..0.8
	res := CSGBoolean(a, b, CSGUnion, mesh.Col(100, 100, 100), mesh.Col(200, 200, 200))
	if res == nil || len(res.Faces) == 0 {
		t.Fatal("CSG union produced empty result")
	}
	if len(res.Faces) < 20 {
		t.Fatalf("union should keep both surfaces, got %d faces", len(res.Faces))
	}
	// 从并集内部 (-0.6,0,0) 沿 +X：第一个面应在 a 的 x=0.1 面（dist 0.7）
	// 但 a 的 x=0.1 面在 b 内部（b min x=-0.2）→ 被丢弃 → 实际第一个面在 b 的 x=0.8 面（dist 1.4）
	d := raycastFirst(res, math3d.Vec3{-0.6, 0, 0}, math3d.Vec3{1, 0, 0})
	if d < 0 || d > 1.6 {
		t.Fatalf("union right wall should be near x=0.8 (dist ~1.4), got %v", d)
	}
	// 从外部 (-2,0,0) 沿 +X：第一个面在 a 的 x=-0.9 面（dist ~1.1）
	d2 := raycastFirst(res, math3d.Vec3{-2, 0, 0}, math3d.Vec3{1, 0, 0})
	if d2 < 0.9 || d2 > 1.3 {
		t.Fatalf("union left wall should be near x=-0.9 (dist ~1.1), got %v", d2)
	}
}

func translateMesh(m *mesh.Mesh, d math3d.Vec3) *mesh.Mesh {
	out := &mesh.Mesh{Positions: make([]math3d.Vec3, len(m.Positions)), Faces: m.Faces}
	for i, p := range m.Positions {
		out.Positions[i] = p.Add(d)
	}
	return out
}

// ---------- 骨骼 ----------

func TestSkeletonWeights(t *testing.T) {
	s := NewSkeleton()
	root := s.AddBone("root", -1, math3d.Vec3{0, 0, 0})
	_ = root
	s.AddBone("head", 0, math3d.Vec3{0, 1, 0})
	// 网格：两个点，一个在 root 旁，一个在 head 旁
	m := &mesh.Mesh{
		Positions: []math3d.Vec3{{0, 0.05, 0}, {0, 0.95, 0}},
		Faces:     []mesh.Face{{A: 0, B: 1, C: 1, Col: mesh.Col(1, 1, 1)}},
	}
	ws := s.SkinWeights(m, 0)
	if len(ws) != 2 {
		t.Fatal("weights count wrong")
	}
	// 顶点 0 靠近 root（骨骼 0），顶点 1 靠近 head（骨骼 1）
	if ws[0].Bones[0] != 0 {
		t.Fatalf("v0 should bind to root bone, got %d", ws[0].Bones[0])
	}
	if ws[1].Bones[0] != 1 {
		t.Fatalf("v1 should bind to head bone, got %d", ws[1].Bones[0])
	}
}

func TestSkinMeshMoves(t *testing.T) {
	s := NewSkeleton()
	s.AddBone("root", -1, math3d.Vec3{0, 0, 0})
	// 网格顶点在 root 附近
	m := &mesh.Mesh{
		Positions: []math3d.Vec3{{0, 0.1, 0}},
		Faces:     []mesh.Face{{A: 0, B: 0, C: 0, Col: mesh.Col(1, 1, 1)}},
	}
	ws := s.SkinWeights(m, 0)
	// 移动 root
	s.Bones[0].Pos = math3d.Vec3{5, 0, 0}
	skinned := s.SkinMesh(m, ws)
	if skinned.Positions[0].X < 4.9 {
		t.Fatalf("vertex should follow bone, got %v", skinned.Positions[0])
	}
}

func TestAnimationSample(t *testing.T) {
	a := &Animation{Name: "test", Loop: true}
	a.AddKey(0, 0, math3d.Vec3{0, 0, 0}, 0, 0, 0)
	a.AddKey(0, 1, math3d.Vec3{0, 2, 0}, 0, 0, 0)
	// 中点插值
	p, _, _, _ := a.Sample(0, 0.5)
	if p.Y != 1 {
		t.Fatalf("mid sample should be y=1, got %v", p)
	}
	// 越界
	p, _, _, _ = a.Sample(0, 5)
	if p.Y != 2 {
		t.Fatalf("beyond last key should clamp, got %v", p)
	}
}

// ---------- OBJ ----------

func TestOBJRoundTrip(t *testing.T) {
	m := mesh.Cube(1.0)
	path := filepath.Join(t.TempDir(), "cube.obj")
	if err := SaveOBJ(path, m); err != nil {
		t.Fatal("save:", err)
	}
	m2, err := LoadOBJ(path)
	if err != nil {
		t.Fatal("load:", err)
	}
	if len(m2.Positions) != len(m.Positions) {
		t.Fatalf("vertex count mismatch: %d vs %d", len(m2.Positions), len(m.Positions))
	}
	if len(m2.Faces) != len(m.Faces) {
		t.Fatalf("face count mismatch: %d vs %d", len(m2.Faces), len(m.Faces))
	}
	// 体积近似
	v1 := meshVolume(m)
	v2 := meshVolume(m2)
	if v1 < 0.9 || v2 < 0.9 || v1 > 1.1 || v2 > 1.1 {
		t.Fatalf("cube volume should be ~1: %v %v", v1, v2)
	}
	_ = os.Remove(path)
}
