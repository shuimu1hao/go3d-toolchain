package app

import (
	"math"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/mesh"
)

// meshCol 便捷构造 mesh.Color。
func meshCol(r, g, b uint8) mesh.Color { return mesh.Col(r, g, b) }

// Gizmo 轴定义：方向 + 颜色。
type gizmoAxis struct {
	dir math3d.Vec3
	col engine.Color
}

var gizmoAxes = []gizmoAxis{
	{math3d.Vec3{1, 0, 0}, engine.Color{R: 220, G: 70, B: 70}},  // X 红
	{math3d.Vec3{0, 1, 0}, engine.Color{R: 80, G: 200, B: 90}},   // Y 绿
	{math3d.Vec3{0, 0, 1}, engine.Color{R: 80, G: 130, B: 240}},  // Z 蓝
}

// 拖拽状态常量
const (
	DragNone = 0
	// 移动：1=X 2=Y 3=Z
	DragMoveX = 1 + iota
	DragMoveY
	DragMoveZ
	// 旋转：4=X 5=Y 6=Z
	DragRotX
	DragRotY
	DragRotZ
	// 缩放：7=X 8=Y 9=Z 10=等比
	DragScaleX
	DragScaleY
	DragScaleZ
	DragScaleAll
)

// 吸附类型位标志（CAD OSNAP 风格，可多选）。
const (
	SnapGrid   = 1 << iota // 网格捕捉
	SnapVertex             // 端点捕捉
	SnapMid                // 中点捕捉
	SnapCenter             // 圆心/中心捕捉
)

// gizmoHit 返回视口内鼠标点击命中的 gizmo 部件（未命中返回 DragNone）。
// 返回 (mode, 轴索引) 编码为 Drag 常量。
func (e *Editor) gizmoHit(vx, vy int) int {
	if e.editMode != EditBone && (e.sel < 0 || e.sel >= len(e.doc.Objs)) {
		return DragNone
	}
	center, ok := e.gizmoCenter()
	if !ok {
		return DragNone
	}
	cx, cy, ok := e.project(center)
	if !ok {
		return DragNone
	}
	fx, fy := float32(vx), float32(vy)
	dc := math.Sqrt(float64((fx-cx)*(fx-cx) + (fy-cy)*(fy-cy)))
	if dc > 160 {
		return DragNone
	}
	// 等比缩放中心手柄
	if e.mode == ModeScale && dc < 14 {
		return DragScaleAll
	}
	const headR = 10.0
	for i, a := range gizmoAxes {
		// 端点（世界坐标 中心 + 轴*len）
		ep := center.Add(a.dir.MulScalar(e.gizmoLen(nil)))
		ex, ey, ok2 := e.project(ep)
		if !ok2 {
			continue
		}
		switch e.mode {
		case ModeMove:
			// 命中箭头头：端点附近
			if math.Sqrt(float64((fx-ex)*(fx-ex)+(fy-ey)*(fy-ey))) < headR {
				return DragMoveX + i
			}
			// 命中轴线段：到线段距离 < 8
			if distToSeg(fx, fy, cx, cy, ex, ey) < 8 {
				return DragMoveX + i
			}
		case ModeRotate:
			// 旋转环：圆环在屏幕上的投影椭圆（简化：命中圆环上的点）
			if e.hitRotRing(fx, fy, i) {
				return DragRotX + i
			}
		case ModeScale:
			if math.Sqrt(float64((fx-ex)*(fx-ex)+(fy-ey)*(fy-ey))) < headR {
				return DragScaleX + i
			}
			// 命中轴线段（同移动模式）
			if distToSeg(fx, fy, cx, cy, ex, ey) < 8 {
				return DragScaleX + i
			}
		}
	}
	return DragNone
}

// distToSeg 点到线段距离。
func distToSeg(px, py, ax, ay, bx, by float32) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-9 {
		return math.Sqrt(float64((px-ax)*(px-ax) + (py-ay)*(py-ay)))
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := ax+t*dx, ay+t*dy
	return math.Sqrt(float64((px-cx)*(px-cx) + (py-cy)*(py-cy)))
}

// hitRotRing 判断点是否命中旋转环（环半径 = 中心到端点距离）。
func (e *Editor) hitRotRing(fx, fy float32, axisIdx int) bool {
	center, ok := e.gizmoCenter()
	if !ok {
		return false
	}
	cx, cy, ok := e.project(center)
	if !ok {
		return false
	}
	ep := center.Add(gizmoAxes[axisIdx].dir.MulScalar(e.gizmoLen(nil)))
	ex, ey, ok2 := e.project(ep)
	if !ok2 {
		return false
	}
	r := math.Sqrt(float64((ex-cx)*(ex-cx) + (ey-cy)*(ey-cy)))
	if r < 8 {
		return false
	}
	d := math.Sqrt(float64((fx-cx)*(fx-cx) + (fy-cy)*(fy-cy)))
	return math.Abs(d-r) < 7
}

// gizmoCenter 返回 gizmo 作用的中心（对象位置或骨骼世界位置）。
func (e *Editor) gizmoCenter() (math3d.Vec3, bool) {
	if e.editMode == EditBone {
		return e.boneWorld()
	}
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return math3d.Vec3{}, false
	}
	return e.doc.Objs[e.sel].Pos, true
}

// gizmoLen 返回 gizmo 长度（世界单位，随缩放适配）。
func (e *Editor) gizmoLen(_ *Object) float32 {
	base := float32(1.4)
	center, ok := e.gizmoCenter()
	if !ok {
		return base
	}
	cx, cy, ok := e.project(center)
	if !ok {
		return base
	}
	ep := center.Add(math3d.Vec3{1, 0, 0})
	ex, ey, ok2 := e.project(ep)
	if !ok2 {
		return base
	}
	px := math.Sqrt(float64((ex-cx)*(ex-cx) + (ey-cy)*(ey-cy)))
	if px > 1 {
		return base * 70 / float32(px)
	}
	return base
}

// drawGizmo 在视口上绘制变换 gizmo。
func (e *Editor) drawGizmo(c *engine.Canvas) {
	if e.editMode != EditBone && (e.sel < 0 || e.sel >= len(e.doc.Objs)) {
		return
	}
	center, ok := e.gizmoCenter()
	if !ok {
		return
	}
	cx, cy, ok := e.project(center)
	if !ok {
		return
	}
	px, py := e.vpX+int(cx), e.vpY+int(cy)
	// 中心点
	c.FillRect(px-2, py-2, 5, 5, engine.Color{R: 220, G: 220, B: 220})
	// 等比缩放手柄（缩放模式）
	if e.mode == ModeScale {
		c.Circle(px, py, 7, engine.Color{R: 220, G: 220, B: 120})
	}
	active := e.activeDragAxis()
	for i, a := range gizmoAxes {
		ep := center.Add(a.dir.MulScalar(e.gizmoLen(nil)))
		ex, ey, ok2 := e.project(ep)
		if !ok2 {
			continue
		}
		ex2, ey2 := e.vpX+int(ex), e.vpY+int(ey)
		col := a.col
		if active == i {
			col = engine.Color{R: 255, G: 255, B: 200}
		}
		switch e.mode {
		case ModeMove:
			c.Line(px, py, ex2, ey2, col)
			c.FillRect(ex2-4, ey2-4, 9, 9, col)
		case ModeScale:
			c.Line(px, py, ex2, ey2, col)
			c.FillRect(ex2-3, ey2-3, 7, 7, col)
		case ModeRotate:
			// 画环（采样投影）
			steps := 48
			var prevX, prevY int
			havePrev := false
			r := e.gizmoLen(nil)
			for s := 0; s <= steps; s++ {
				th := 2 * math.Pi * float64(s) / float64(steps)
				// 绕轴旋转的单位圆 → 世界点
				var wp math3d.Vec3
				switch i {
				case 0: // X 轴：YZ 平面圆
					wp = center.Add(math3d.Vec3{0, r * float32(math.Cos(th)), r * float32(math.Sin(th))})
				case 1: // Y 轴：XZ 平面圆
					wp = center.Add(math3d.Vec3{r * float32(math.Cos(th)), 0, r * float32(math.Sin(th))})
				case 2: // Z 轴：XY 平面圆
					wp = center.Add(math3d.Vec3{r * float32(math.Cos(th)), r * float32(math.Sin(th)), 0})
				}
				sxx, syy, ok3 := e.project(wp)
				if !ok3 {
					havePrev = false
					continue
				}
				if havePrev {
					c.Line(prevX, prevY, e.vpX+int(sxx), e.vpY+int(syy), col)
				}
				prevX, prevY = e.vpX+int(sxx), e.vpY+int(syy)
				havePrev = true
			}
		}
	}
}

// activeDragAxis 返回当前拖拽的轴索引（0-2），-1 无。
func (e *Editor) activeDragAxis() int {
	switch e.drag {
	case DragMoveX, DragRotX, DragScaleX:
		return 0
	case DragMoveY, DragRotY, DragScaleY:
		return 1
	case DragMoveZ, DragRotZ, DragScaleZ:
		return 2
	}
	return -1
}

// ---------- 拖拽更新 ----------

// updateDrag 处理 gizmo 拖拽（鼠标移动时调用，mx/my 为视口坐标）。
// 编辑目标：骨骼模式下为选中骨骼，否则为选中对象。
func (e *Editor) updateDrag(mx, my int) {
	if e.drag == DragNone {
		return
	}
	var o *Object
	var bone *Bone
	if e.editMode == EditBone {
		oo := e.selObj()
		if oo != nil && oo.Skeleton != nil && e.selBone >= 0 && e.selBone < len(oo.Skeleton.Bones) {
			o, bone = oo, oo.Skeleton.Bones[e.selBone]
		}
	} else if e.sel >= 0 && e.sel < len(e.doc.Objs) {
		o = e.doc.Objs[e.sel]
	}
	if o == nil {
		e.drag = DragNone
		return
	}
	center, _ := e.gizmoCenter()
	fx, fy := float32(mx), float32(my)
	cam := e.cam.Camera()
	// 相机前向（世界）
	viewDir := cam.Pos.Sub(e.cam.Target).Normalized()
	// 移动/旋转目标引用
	movePos := func() *math3d.Vec3 {
		if bone != nil {
			return &bone.Pos
		}
		return &o.Pos
	}
	rotRef := func() (rx, ry, rz *float32) {
		if bone != nil {
			return &bone.RotX, &bone.RotY, &bone.RotZ
		}
		return &o.RotX, &o.RotY, &o.RotZ
	}
	switch {
	case e.drag >= DragMoveX && e.drag <= DragMoveZ:
		axis := gizmoAxes[e.drag-DragMoveX].dir
		p, ok := e.unproject(fx, fy, center, viewDir)
		if ok && e.dragStartOK {
			delta := p.Sub(e.dragAnchor)
			pos := movePos()
			*pos = pos.Add(axis.MulScalar(delta.Dot(axis)))
			// CAD 风格吸附：端点/中点/圆心优先，网格兜底
			if wp, sn := e.snapWorld(fx, fy, *pos); sn {
				*pos = wp
			}
			e.dragAnchor = p // 增量模式：锚点随拖拽更新
		}
		if ok && !e.dragStartOK {
			e.dragAnchor = p
			e.dragStartOK = true
		}
	case e.drag >= DragRotX && e.drag <= DragRotZ:
		cx, cy, ok := e.project(center)
		if ok {
			ang := float32(math.Atan2(float64(fy-cy), float64(fx-cx)))
			if e.dragStartOK {
				da := ang - e.dragAnchorAng
				rx, ry, rz := rotRef()
				switch e.drag {
				case DragRotX:
					*rx += da
				case DragRotY:
					*ry += da
				case DragRotZ:
					*rz += da
				}
			}
			e.dragAnchorAng = ang
			e.dragStartOK = true
		}
	case e.drag == DragScaleAll:
		cx, cy, ok := e.project(center)
		if ok {
			dc := math.Sqrt(float64((fx-cx)*(fx-cx) + (fy-cy)*(fy-cy)))
			if e.dragStartOK {
				ratio := float32(dc / e.dragAnchorDist)
				if ratio > 0.01 && ratio < 20 {
					o.Scale *= ratio
					if o.Scale < 0.01 {
						o.Scale = 0.01
					}
					if o.Scale > 100 {
						o.Scale = 100
					}
				}
			}
			e.dragAnchorDist = dc
			e.dragStartOK = true
		}
	case e.drag >= DragScaleX && e.drag <= DragScaleZ:
		axis := gizmoAxes[e.drag-DragScaleX].dir
		p, ok := e.unproject(fx, fy, center, viewDir)
		if ok && e.dragStartOK {
			delta := p.Sub(e.dragAnchor).Dot(axis)
			o.Scale += delta * 0.15
			if o.Scale < 0.01 {
				o.Scale = 0.01
			}
			if o.Scale > 100 {
				o.Scale = 100
			}
			e.dragAnchor = p
		}
		if ok && !e.dragStartOK {
			e.dragAnchor = p
			e.dragStartOK = true
		}
	}
}

// ---------- 吸附（CAD OSNAP 风格） ----------

// snapWorld 视口坐标 → 吸附目标世界点。优先级：端点 > 中点 > 圆心 > 网格。
// gridRef 是网格吸附的参考点（当前移动位置）。
func (e *Editor) snapWorld(fx, fy float32, gridRef math3d.Vec3) (math3d.Vec3, bool) {
	if !e.snap {
		return math3d.Vec3{}, false
	}
	if e.snapMask&SnapVertex != 0 {
		if p, ok := e.snapNearVertex(fx, fy); ok {
			return p, true
		}
	}
	if e.snapMask&SnapMid != 0 {
		if p, ok := e.snapNearMid(fx, fy); ok {
			return p, true
		}
	}
	if e.snapMask&SnapCenter != 0 {
		if p, ok := e.snapNearCenter(fx, fy); ok {
			return p, true
		}
	}
	if e.snapMask&SnapGrid != 0 {
		return e.snapGrid(gridRef), true
	}
	return math3d.Vec3{}, false
}

// snapNearVertex 屏幕距离 <14px 的最近顶点（排除自身对象）。
func (e *Editor) snapNearVertex(fx, fy float32) (math3d.Vec3, bool) {
	best := math3d.Vec3{}
	bestD := float32(14 * 14)
	found := false
	for oi, o := range e.doc.Objs {
		if e.editMode != EditBone && oi == e.sel {
			continue
		}
		if !o.Visible {
			continue
		}
		m := o.RenderMesh()
		if m == nil {
			continue
		}
		for _, p := range m.Positions {
			wp := o.TransformPoint(p)
			sx, sy, ok := e.project(wp)
			if !ok {
				continue
			}
			d := (fx-sx)*(fx-sx) + (fy-sy)*(fy-sy)
			if d < bestD {
				bestD = d
				best = wp
				found = true
			}
		}
	}
	return best, found
}

// snapNearMid 屏幕距离 <14px 的最近边中点（排除自身对象）。
func (e *Editor) snapNearMid(fx, fy float32) (math3d.Vec3, bool) {
	best := math3d.Vec3{}
	bestD := float32(14 * 14)
	found := false
	for oi, o := range e.doc.Objs {
		if e.editMode != EditBone && oi == e.sel {
			continue
		}
		if !o.Visible {
			continue
		}
		m := o.RenderMesh()
		if m == nil {
			continue
		}
		for _, f := range m.Faces {
			pairs := [3][2]int{{f.A, f.B}, {f.B, f.C}, {f.C, f.A}}
			for _, pair := range pairs {
				p0 := o.TransformPoint(m.Positions[pair[0]])
				p1 := o.TransformPoint(m.Positions[pair[1]])
				mid := p0.Add(p1).MulScalar(0.5)
				sx, sy, ok := e.project(mid)
				if !ok {
					continue
				}
				d := (fx-sx)*(fx-sx) + (fy-sy)*(fy-sy)
				if d < bestD {
					bestD = d
					best = mid
					found = true
				}
			}
		}
	}
	return best, found
}

// snapNearCenter 屏幕距离 <14px 的最近对象中心（物体 Pos，排除自身）。
func (e *Editor) snapNearCenter(fx, fy float32) (math3d.Vec3, bool) {
	best := math3d.Vec3{}
	bestD := float32(14 * 14)
	found := false
	for oi, o := range e.doc.Objs {
		if e.editMode != EditBone && oi == e.sel {
			continue
		}
		if !o.Visible {
			continue
		}
		wp := o.Pos
		sx, sy, ok := e.project(wp)
		if !ok {
			continue
		}
		d := (fx-sx)*(fx-sx) + (fy-sy)*(fy-sy)
		if d < bestD {
			bestD = d
			best = wp
			found = true
		}
	}
	return best, found
}

// snapGrid 网格吸附（round 到 snapStep 倍数）。
func (e *Editor) snapGrid(p math3d.Vec3) math3d.Vec3 {
	s := e.snapStep
	if s <= 0 {
		s = 0.5
	}
	return math3d.Vec3{
		X: float32(math.Round(float64(p.X/s))) * s,
		Y: float32(math.Round(float64(p.Y/s))) * s,
		Z: float32(math.Round(float64(p.Z/s))) * s,
	}
}

// snap1 一维网格吸附（草图 2D 用）。
func (e *Editor) snap1(v float32) float32 {
	s := e.snapStep
	if s <= 0 {
		s = 0.5
	}
	return float32(math.Round(float64(v/s))) * s
}
