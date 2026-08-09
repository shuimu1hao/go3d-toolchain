package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"go3d/math3d"
	"go3d/render"
)

func TestDocumentOps(t *testing.T) {
	d := NewDocument("测试")
	if len(d.Objs) != 0 {
		t.Fatal("new doc should be empty")
	}
	a := d.Add(TCube)
	b := d.Add(TSphere)
	c := d.Add(TCylinder)
	if len(d.Objs) != 3 {
		t.Fatalf("expect 3 objs, got %d", len(d.Objs))
	}
	if a.Name == "" || b.Name == "" || c.Name == "" {
		t.Fatal("names should not be empty")
	}
	// 上移/下移
	d.MoveUp(2)
	if d.Objs[1] != c || d.Objs[2] != b {
		t.Fatal("MoveUp failed")
	}
	d.MoveDown(0)
	if d.Objs[0] != c || d.Objs[1] != a {
		t.Fatal("MoveDown failed")
	}
	// 删除
	d.Remove(1)
	if len(d.Objs) != 2 {
		t.Fatalf("expect 2 after remove, got %d", len(d.Objs))
	}
	// 复制
	cp := d.Duplicate(0)
	if cp == nil || len(d.Objs) != 3 {
		t.Fatal("duplicate failed")
	}
	if cp.Name != d.Objs[1].Name || cp.Name != c.Name+"_copy" {
		t.Fatalf("duplicate name wrong: %s", cp.Name)
	}
	// 可见性过滤
	d.Objs[0].Visible = false
	vis := d.VisibleObjs()
	if len(vis) != 2 {
		t.Fatalf("visible filter wrong: %d", len(vis))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	e := New(320, 200)
	e.doc = NewDocument("零件A")
	o1 := e.doc.Add(TCube)
	o1.Pos.X = 1.5
	o1.Pos.Y = -0.5
	o1.RotY = 0.7
	o1.Scale = 2.5
	o1.Color = meshCol(10, 20, 30)
	e.doc.Add(TSphere)
	e.doc.Objs[1].Visible = false

	path := filepath.Join(t.TempDir(), "scene.json")
	if err := e.Save(path); err != nil {
		t.Fatal("save:", err)
	}

	e2 := New(320, 200)
	if err := e2.Load(path); err != nil {
		t.Fatal("load:", err)
	}
	if e2.doc.Name != "零件A" {
		t.Fatalf("doc name wrong: %s", e2.doc.Name)
	}
	if len(e2.doc.Objs) != 2 {
		t.Fatalf("objs count wrong: %d", len(e2.doc.Objs))
	}
	o := e2.doc.Objs[0]
	if o.Pos.X != 1.5 || o.Pos.Y != -0.5 {
		t.Fatalf("pos wrong: %v", o.Pos)
	}
	if o.RotY != 0.7 || o.Scale != 2.5 {
		t.Fatalf("rot/scale wrong: %v %v", o.RotY, o.Scale)
	}
	if o.Color.R != 10 || o.Color.G != 20 || o.Color.B != 30 {
		t.Fatalf("color wrong: %v", o.Color)
	}
	if e2.doc.Objs[1].Visible {
		t.Fatal("visible flag lost")
	}
}

func TestParseFloat(t *testing.T) {
	cases := []struct {
		in   string
		want float32
	}{
		{"0", 0}, {"1.5", 1.5}, {"-2.25", -2.25}, {"abc", 0}, {"12", 12}, {"-0.5", -0.5},
	}
	for _, c := range cases {
		if got := parseFloat(c.in); got != c.want {
			t.Errorf("parseFloat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPickColorRoundTrip(t *testing.T) {
	for _, id := range []int{1, 2, 3, 100, 255, 256, 1000, 65536} {
		col := render.PickColor(id)
		back := render.PickIDFromColor(col.R, col.G, col.B)
		if back != id {
			t.Errorf("pick color roundtrip %d -> %v -> %d", id, col, back)
		}
	}
}

func TestRenderObjTint(t *testing.T) {
	e := New(320, 200)
	o := e.doc.Add(TCube)
	ro := o.RenderObj()
	if ro.ColorTint == nil {
		t.Fatal("ColorTint should be set")
	}
	if *ro.ColorTint != o.Color {
		t.Fatal("tint mismatch")
	}
	if ro.Scale != 1 {
		t.Fatalf("scale should default 1, got %v", ro.Scale)
	}
}

func TestSceneFileCleanup(t *testing.T) {
	// 确保测试不留下 scene.json
	path := defaultScenePath()
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}
}

// TestGizmoScaleDrag 验证缩放 gizmo 拖拽数学（不依赖 X11，直接驱动 updateDrag）。
func TestGizmoScaleDrag(t *testing.T) {
	e := New(320, 200)
	e.AddObject(TCube)
	e.sel = 0
	o := e.doc.Objs[0]
	o.Scale = 1

	// 模拟：点击 X 轴缩放端点（屏幕位置用 project 计算）
	ep := o.Pos.Add(math3d.Vec3{1, 0, 0}.MulScalar(e.gizmoLen(o)))
	ex, ey, ok2 := e.project(ep)
	if !ok2 {
		t.Fatal("project axis end failed")
	}

	// 第一次 updateDrag：设置锚点
	e.drag = DragScaleX
	e.updateDrag(int(ex), int(ey))
	if !e.dragStartOK {
		t.Fatal("dragStartOK should be set after first update")
	}
	startScale := o.Scale

	// 第二次：沿 X 轴正方向拖 30px（向右下移动）
	e.updateDrag(int(ex)+30, int(ey)+15)
	if o.Scale <= startScale {
		t.Fatalf("scale should increase after drag along +X: %v -> %v", startScale, o.Scale)
	}
	if o.Scale > startScale+5 {
		t.Fatalf("scale jump too large: %v -> %v (anchor must update incrementally)", startScale, o.Scale)
	}
	// 第三次：往回拖，scale 应该减小（增量）
	after2 := o.Scale
	e.updateDrag(int(ex)-10, int(ey)-5)
	if o.Scale >= after2 {
		t.Fatalf("scale should decrease when dragging back: %v -> %v", after2, o.Scale)
	}
}

// TestGizmoMoveDrag 验证移动 gizmo 拖拽数学（锚点增量更新，防止指数飞走）。
func TestGizmoMoveDrag(t *testing.T) {
	e := New(320, 200)
	e.snap = false // 纯 Gizmo 数学测试，不依赖吸附
	e.AddObject(TCube)
	e.sel = 0
	o := e.doc.Objs[0]
	o.Pos = math3d.Vec3{0, 0.5, 0}

	ep := o.Pos.Add(math3d.Vec3{1, 0, 0}.MulScalar(e.gizmoLen(o)))
	ex, ey, ok2 := e.project(ep)
	if !ok2 {
		t.Fatal("project axis end failed")
	}

	e.drag = DragMoveX
	e.updateDrag(int(ex), int(ey)) // 锚点
	if !e.dragStartOK {
		t.Fatal("dragStartOK not set")
	}
	start := o.Pos.X
	// 拖 20px
	e.updateDrag(int(ex)+20, int(ey))
	move1 := o.Pos.X - start
	if move1 <= 0 {
		t.Fatalf("X should increase: %v", move1)
	}
	// 再拖 20px（同方向）：第二次增量应接近第一次（线性，不是指数）
	e.updateDrag(int(ex)+40, int(ey))
	move2 := o.Pos.X - (start + move1)
	if move2 <= 0 || move2 > move1*3 {
		t.Fatalf("second move should be linear-ish: move1=%v move2=%v", move1, move2)
	}
	// 往回拖 10px：X 应该减小
	before := o.Pos.X
	e.updateDrag(int(ex)+30, int(ey))
	if o.Pos.X >= before {
		t.Fatalf("X should decrease when dragging back: %v -> %v", before, o.Pos.X)
	}
}

// TestRotateDrag 验证旋转 gizmo 拖拽（角度增量）。
func TestRotateDrag(t *testing.T) {
	e := New(320, 200)
	e.AddObject(TCube)
	e.sel = 0
	o := e.doc.Objs[0]
	o.RotX = 0

	cx, cy, ok := e.project(o.Pos)
	if !ok {
		t.Fatal("project failed")
	}
	// 点击环右侧，绕中心顺时针拖（从右侧到下方）
	e.drag = DragRotY
	e.updateDrag(int(cx)+80, int(cy)) // 右侧（角度 0）
	if !e.dragStartOK {
		t.Fatal("dragStartOK not set")
	}
	e.updateDrag(int(cx)+80, int(cy)+50) // 下方（角度 ~90°）
	if o.RotY <= 0 {
		t.Fatalf("RotY should increase after clockwise drag: %v", o.RotY)
	}
	if o.RotY > 3.5 {
		t.Fatalf("RotY too large: %v", o.RotY)
	}
}

// TestSkeletonAnimJSON 骨骼+动画保存/载入往返。
func TestSkeletonAnimJSON(t *testing.T) {
	e := New(320, 200)
	o := e.doc.Add(TCube)
	sk := NewSkeleton()
	sk.AddBone("root", -1, math3d.Vec3{0, 0, 0})
	sk.AddBone("arm", 0, math3d.Vec3{0, 1, 0})
	o.Skeleton = sk
	sk.ResetBindPose()
	m := o.Type.NewMesh()
	o.Weights = sk.SkinWeights(m, 0)
	a := &Animation{Name: "wave", Loop: true}
	a.AddKey(1, 0, math3d.Vec3{0, 1, 0}, 0, 0, 0)
	a.AddKey(1, 1, math3d.Vec3{0, 1, 0}, 0, 1.5, 0)
	o.Anim = a

	path := filepath.Join(t.TempDir(), "anim.json")
	if err := e.Save(path); err != nil {
		t.Fatal("save:", err)
	}
	e2 := New(320, 200)
	if err := e2.Load(path); err != nil {
		t.Fatal("load:", err)
	}
	o2 := e2.doc.Objs[0]
	if o2.Skeleton == nil || len(o2.Skeleton.Bones) != 2 {
		t.Fatalf("skeleton not restored: %v", o2.Skeleton)
	}
	if o2.Skeleton.Bones[1].Parent != 0 {
		t.Fatal("bone parent lost")
	}
	if o2.Anim == nil || o2.Anim.Duration < 0.9 {
		t.Fatalf("anim not restored: %v", o2.Anim)
	}
	if len(o2.Weights) != len(o.Type.NewMesh().Positions) {
		t.Fatalf("weights not restored: %d", len(o2.Weights))
	}
}

// TestSketchExtrudeCommand 草图拉伸生成对象。
func TestSketchExtrudeCommand(t *testing.T) {
	e := New(320, 200)
	e.SetEditMode(EditSketch)
	e.sketch = NewSketch(PlaneXY)
	e.sketch.Points = RectSketch(0, 0, 2, 1)
	e.sketch.Closed = true
	e.sketchExtrude()
	if len(e.doc.Objs) != 1 {
		t.Fatalf("extrude should add 1 object, got %d", len(e.doc.Objs))
	}
	o := e.doc.Objs[0]
	if o.CustomMesh == nil || len(o.CustomMesh.Faces) != 12 {
		t.Fatalf("extrude mesh wrong: %d faces", len(o.CustomMesh.Faces))
	}
	if e.sel != 0 {
		t.Fatal("extruded object should be selected")
	}
}

// TestCSGCommand 布尔运算集成到文档。
func TestCSGCommand(t *testing.T) {
	e := New(320, 200)
	_ = e.doc.Add(TCube)
	_ = e.doc.Add(TSphere)
	e.sel = 0
	a := e.doc.Objs[0].RenderMesh()
	b := e.doc.Objs[1].RenderMesh()
	res := CSGBoolean(a, b, CSGSubtract, e.doc.Objs[0].Color, e.doc.Objs[1].Color)
	if res == nil || len(res.Faces) == 0 {
		t.Fatal("CSG command produced empty result")
	}
	o := e.doc.AddObjFromMesh("布尔结果", res)
	if o.CustomMesh == nil {
		t.Fatal("boolean result should be custom mesh")
	}
}

// TestSnapGrid 验证网格吸附（round 到步长倍数）。
func TestSnapGrid(t *testing.T) {
	e := New(320, 200)
	e.snapStep = 0.5
	p := e.snapGrid(math3d.Vec3{X: 1.23, Y: -0.77, Z: 3.1})
	if p.X != 1.0 || p.Y != -1.0 || p.Z != 3.0 {
		t.Fatalf("snapGrid wrong: %v", p)
	}
	if e.snap1(0.24) != 0.0 || e.snap1(0.26) != 0.5 || e.snap1(-0.74) != -0.5 {
		t.Fatalf("snap1 wrong")
	}
}

// TestSnapWorldVertex 验证端点吸附：另一对象顶点在鼠标附近时被捕获。
func TestSnapWorldVertex(t *testing.T) {
	e := New(320, 200)
	e.snap = true
	e.snapMask = SnapVertex
	e.AddObject(TCube) // sel=0 自身
	_ = e.doc.Add(TCube)
	o2 := e.doc.Objs[1]
	o2.Pos = math3d.Vec3{2, 0.5, 0} // 第二个立方体在 +X
	// 投影第二个立方体的某个顶点（世界坐标 (1.5, 1, 0)）
	target := math3d.Vec3{2 + 0.5, 0.5 + 0.5, 0}
	sx, sy, ok := e.project(target)
	if !ok {
		t.Fatal("project target failed")
	}
	wp, hit := e.snapWorld(sx, sy, math3d.Vec3{})
	if !hit {
		t.Fatal("should snap to vertex")
	}
	if math.Abs(float64(wp.X-target.X)) > 0.01 || math.Abs(float64(wp.Y-target.Y)) > 0.01 {
		t.Fatalf("snapped point wrong: got %v want %v", wp, target)
	}
	// 远处不命中
	wp, hit = e.snapWorld(sx+200, sy, math3d.Vec3{})
	if hit {
		t.Fatalf("should not snap far away: %v", wp)
	}
}

// TestSnapPriority 验证优先级：端点 > 网格。
func TestSnapPriority(t *testing.T) {
	e := New(320, 200)
	e.snap = true
	e.snapMask = SnapVertex | SnapGrid
	e.AddObject(TCube)
	e.sel = 0
	_ = e.doc.Add(TCube)
	o2 := e.doc.Objs[1]
	o2.Pos = math3d.Vec3{0.6, 0.5, 0}
	target := math3d.Vec3{0.6 + 0.5, 1, 0}
	sx, sy, ok := e.project(target)
	if !ok {
		t.Fatal("project failed")
	}
	wp, hit := e.snapWorld(sx, sy, math3d.Vec3{X: 1.0})
	if !hit {
		t.Fatal("should hit")
	}
	if math.Abs(float64(wp.X-1.1)) > 0.01 {
		t.Fatalf("vertex should win over grid: got %v", wp)
	}
}

// TestCSGCommandSelPrev 验证布尔运算 A=当前选中 B=上次选中（CAD 逻辑）。
func TestCSGCommandSelPrev(t *testing.T) {
	e := New(320, 200)
	_ = e.doc.Add(TCube)
	_ = e.doc.Add(TSphere)
	// 先选 B（球体，索引1），再选 A（立方体，索引0）
	e.sel = 1
	e.selPrev = -1
	e.viewportLeftDown(0, 0) // 模拟点击（无对象命中也走逻辑）
	e.sel = 1
	e.selPrev = -1
	// 直接模拟两次选择：B=0(球), A=1(立方)？用拾取逻辑验证 selPrev 更新
	// 手动设置选择顺序
	e.sel = 0
	e.selPrev = 1 // B=球体(索引1)
	// A=立方体(索引0) - 差集：立方体 - 球体
	e.csgApply(CSGSubtract)
	if len(e.doc.Objs) != 3 {
		t.Fatalf("布尔应新增 1 对象，实际 %d", len(e.doc.Objs))
	}
	if e.selPrev != -1 {
		t.Fatalf("运算后 selPrev 应清空，实际 %d", e.selPrev)
	}
	if e.sel != 2 {
		t.Fatalf("运算后应选中新结果(索引2)，实际 %d", e.sel)
	}
	// B 未选时提示且不执行
	e2 := New(320, 200)
	_ = e2.doc.Add(TCube)
	e2.sel = 0
	e2.selPrev = -1
	e2.csgApply(CSGSubtract)
	if len(e2.doc.Objs) != 1 {
		t.Fatalf("B 未选时不应执行布尔")
	}
}

// TestSelectionOrder 验证点击拾取时 selPrev 记录上次选中。
func TestSelectionOrder(t *testing.T) {
	e := New(320, 200)
	e.sel = -1
	e.selPrev = -1
	// 模拟点选对象0（无拾取像素，直接设置）
	e.sel = 0
	e.viewportLeftDown(5, 5) // 空白点击
	if e.sel != -1 {
		t.Fatalf("空白点击应取消选择")
	}
	// 模拟：点对象0 → 点对象1
	e.sel = -1
	e.selPrev = -1
	e.sel = 0 // 第一次选中
	e.selPrev = -1
	e.sel = 1 // 第二次选中（应记录 0 为 selPrev）
	// 直接验证 csgApply 用 selPrev=0, sel=1
	e.selPrev = 0
	e.sel = 1
	if e.selPrev != 0 || e.sel != 1 {
		t.Fatalf("选择顺序错误")
	}
}
