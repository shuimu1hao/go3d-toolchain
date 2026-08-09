package app

import (
	"math"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/render"
)

// drawViewport 渲染 3D 视口。
func (e *Editor) drawViewport(c *engine.Canvas) {
	e.vpX = TreeW
	e.vpY = ToolbarH
	e.vpW = c.W - TreeW - PropW
	e.vpH = c.H - ToolbarH - StatusH
	vw, vh := e.vpW, e.vpH

	// 像素缓冲
	if len(e.vpPixels) != vw*vh*4 {
		e.vpPixels = make([]byte, vw*vh*4)
		e.rd = render.NewRenderer(vw, vh)
	}
	e.rd.Clear(e.vpPixels)
	objs := e.renderObjects()
	e.rd.Render(e.vpPixels, e.cam.Camera(), objs)
	// 上屏
	for y := 0; y < vh; y++ {
		row := y * vw * 4
		for x := 0; x < vw; x++ {
			i := row + x*4
			c.SetPixel(x+e.vpX, y+e.vpY, engine.Color{R: e.vpPixels[i+2], G: e.vpPixels[i+1], B: e.vpPixels[i]})
		}
	}

	// 精灵公告板（2D 叠加）
	for _, i := range e.level.Insts {
		if !i.Visible || !i.IsSprite || i.Sprite == nil {
			continue
		}
		sx, sy, ok := e.project(i.Pos)
		if ok {
			sz := int(e.spriteSize * i.Scale * float32(e.vpW) / 6)
			if sz > 2 {
				DrawSprite(c, i.Sprite, int(sx)+e.vpX, int(sy)+e.vpY, sz, sz)
			}
		}
	}

	// 网格 + gizmo + 选中框
	e.drawGridOverlay(c)
	if !e.playing {
		e.drawGizmo(c)
		if e.sel >= 0 && e.sel < len(e.level.Insts) {
			e.drawSelectionBox(c)
		}
	}
}

// viewProj 视图投影矩阵（与建模编辑器一致）。
func (e *Editor) viewProj() math3d.Mat4 {
	cam := e.cam.Camera()
	aspect := float32(e.vpW) / float32(e.vpH)
	var proj math3d.Mat4
	if cam.Ortho {
		size := cam.OrthoSize
		if size <= 0 {
			size = 8
		}
		proj = math3d.Ortho(-size*aspect/2, size*aspect/2, -size/2, size/2, cam.Near, cam.Far)
	} else {
		proj = math3d.Perspective(cam.FOV, aspect, cam.Near, cam.Far)
	}
	var view math3d.Mat4
	if cam.LookTarget != nil {
		view = math3d.LookAt(cam.Pos, *cam.LookTarget, math3d.Vec3{0, 1, 0})
	} else {
		view = math3d.FPSView(cam.Pos, cam.Yaw, cam.Pitch)
	}
	return math3d.Mul(proj, view)
}

// project 世界点 → 视口内坐标。
func (e *Editor) project(wp math3d.Vec3) (sx, sy float32, ok bool) {
	vp := e.viewProj()
	x, y, z, w := vp.TransformW(wp)
	if w <= 1e-9 {
		return 0, 0, false
	}
	ndcX, ndcY, ndcZ := x/w, y/w, z/w
	if ndcZ < -1 || ndcZ > 1 {
		return 0, 0, false
	}
	sx = (ndcX + 1) * 0.5 * float32(e.vpW)
	sy = (1 - (ndcY+1)*0.5) * float32(e.vpH)
	return sx, sy, true
}

// unproject 视口坐标 → 「过 anchor、法线 viewDir 的平面」上的世界点。
func (e *Editor) unproject(sx, sy float32, anchor, viewDir math3d.Vec3) (math3d.Vec3, bool) {
	ndcX := sx/float32(e.vpW)*2 - 1
	ndcY := 1 - sy/float32(e.vpH)*2
	vp := e.viewProj()
	pNear, ok1 := vp.Unproject(ndcX, ndcY, 0)
	pFar, ok2 := vp.Unproject(ndcX, ndcY, 1)
	if !ok1 || !ok2 {
		return math3d.Vec3{}, false
	}
	dir := pFar.Sub(pNear).Normalized()
	if math.Abs(float64(dir.Dot(viewDir))) < 1e-6 {
		return math3d.Vec3{}, false
	}
	t := anchor.Sub(pNear).Dot(viewDir) / dir.Dot(viewDir)
	return pNear.Add(dir.MulScalar(t)), true
}

// drawGridOverlay 网格线（XZ 平面）。
func (e *Editor) drawGridOverlay(c *engine.Canvas) {
	col := engine.Color{R: 52, G: 58, B: 70}
	step := float32(1)
	if e.cam.Dist > 12 {
		step = 2
	}
	if e.cam.Dist > 25 {
		step = 5
	}
	for gx := -20; gx <= 20; gx += int(step) {
		ax, ay, ok1 := e.project(math3d.Vec3{float32(gx), 0, -20})
		bx, by, ok2 := e.project(math3d.Vec3{float32(gx), 0, 20})
		if ok1 && ok2 {
			c.Line(int(ax)+e.vpX, int(ay)+e.vpY, int(bx)+e.vpX, int(by)+e.vpY, col)
		}
	}
	for gz := -20; gz <= 20; gz += int(step) {
		ax, ay, ok1 := e.project(math3d.Vec3{-20, 0, float32(gz)})
		bx, by, ok2 := e.project(math3d.Vec3{20, 0, float32(gz)})
		if ok1 && ok2 {
			c.Line(int(ax)+e.vpX, int(ay)+e.vpY, int(bx)+e.vpX, int(by)+e.vpY, col)
		}
	}
	ox, oy, _ := e.project(math3d.Vec3{0, 0.01, 0})
	xe, ye, _ := e.project(math3d.Vec3{1, 0.01, 0})
	_ = ox
	_ = oy
	_ = xe
	_ = ye
	c.Line(int(ox)+e.vpX, int(oy)+e.vpY, int(xe)+e.vpX, int(ye)+e.vpY, engine.Color{R: 220, G: 70, B: 70})
	xe, ye, _ = e.project(math3d.Vec3{0, 1.01, 0})
	c.Line(int(ox)+e.vpX, int(oy)+e.vpY, int(xe)+e.vpX, int(ye)+e.vpY, engine.Color{R: 80, G: 200, B: 90})
	xe, ye, _ = e.project(math3d.Vec3{0, 0.01, 1})
	c.Line(int(ox)+e.vpX, int(oy)+e.vpY, int(xe)+e.vpX, int(ye)+e.vpY, engine.Color{R: 70, G: 120, B: 230})
}

// drawSelectionBox 选中实例包围盒线框。
func (e *Editor) drawSelectionBox(c *engine.Canvas) {
	inst := e.level.Insts[e.sel]
	if inst.IsSprite || inst.Res == nil || inst.Res.Mesh == nil {
		return
	}
	m := inst.Res.Mesh
	if m == nil || len(m.Positions) == 0 {
		return
	}
	min := math3d.Vec3{1e9, 1e9, 1e9}
	max := math3d.Vec3{-1e9, -1e9, -1e9}
	sc := inst.Scale
	for _, p := range m.Positions {
		w := inst.Pos.Add(math3d.Vec3{p.X * sc, p.Y * sc, p.Z * sc})
		if w.X < min.X {
			min.X = w.X
		}
		if w.Y < min.Y {
			min.Y = w.Y
		}
		if w.Z < min.Z {
			min.Z = w.Z
		}
		if w.X > max.X {
			max.X = w.X
		}
		if w.Y > max.Y {
			max.Y = w.Y
		}
		if w.Z > max.Z {
			max.Z = w.Z
		}
	}
	col := engine.Color{R: 255, G: 200, B: 60}
	corners := [8]math3d.Vec3{
		{min.X, min.Y, min.Z}, {max.X, min.Y, min.Z}, {max.X, max.Y, min.Z}, {min.X, max.Y, min.Z},
		{min.X, min.Y, max.Z}, {max.X, min.Y, max.Z}, {max.X, max.Y, max.Z}, {min.X, max.Y, max.Z},
	}
	edges := [12][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 0}, {4, 5}, {5, 6}, {6, 7}, {7, 4}, {0, 4}, {1, 5}, {2, 6}, {3, 7}}
	for _, ed := range edges {
		p1x, p1y, ok1 := e.project(corners[ed[0]])
		p2x, p2y, ok2 := e.project(corners[ed[1]])
		if ok1 && ok2 {
			c.Line(int(p1x)+e.vpX, int(p1y)+e.vpY, int(p2x)+e.vpX, int(p2y)+e.vpY, col)
		}
	}
}

// drawGizmo 变换 gizmo。
func (e *Editor) drawGizmo(c *engine.Canvas) {
	if e.sel < 0 || e.sel >= len(e.level.Insts) {
		return
	}
	inst := e.level.Insts[e.sel]
	if inst.IsSprite {
		return
	}
	center := inst.Pos
	cx, cy, ok := e.project(center)
	if !ok {
		return
	}
	axes := []struct {
		dir math3d.Vec3
		col engine.Color
	}{
		{math3d.Vec3{1, 0, 0}, engine.Color{R: 220, G: 70, B: 70}},
		{math3d.Vec3{0, 1, 0}, engine.Color{R: 80, G: 200, B: 90}},
		{math3d.Vec3{0, 0, 1}, engine.Color{R: 70, G: 120, B: 230}},
	}
	ex0, ey0, okE := e.project(center.Add(math3d.Vec3{1, 0, 0}))
	if !okE {
		return
	}
	pxlen := math.Sqrt(float64((ex0-cx)*(ex0-cx) + (ey0-cy)*(ey0-cy)))
	ln := 70.0
	if pxlen > 1 {
		ln = 70.0 / pxlen
	}
	for _, ax := range axes {
		ep := center.Add(ax.dir.MulScalar(float32(ln)))
		ex, ey, ok2 := e.project(ep)
		if !ok2 {
			continue
		}
		c.Line(int(cx)+e.vpX, int(cy)+e.vpY, int(ex)+e.vpX, int(ey)+e.vpY, ax.col)
		c.FillRect(int(ex)+e.vpX-3, int(ey)+e.vpY-3, 7, 7, ax.col)
	}
}

// gizmoHit 命中测试（视口内坐标），返回轴索引或 -1。
func (e *Editor) gizmoHit(vx, vy int) int {
	if e.sel < 0 || e.sel >= len(e.level.Insts) {
		return -1
	}
	inst := e.level.Insts[e.sel]
	if inst.IsSprite {
		return -1
	}
	center := inst.Pos
	cx, cy, ok := e.project(center)
	if !ok {
		return -1
	}
	fx, fy := float32(vx), float32(vy)
	ex0, ey0, okE := e.project(center.Add(math3d.Vec3{1, 0, 0}))
	if !okE {
		return -1
	}
	pxlen := math.Sqrt(float64((ex0-cx)*(ex0-cx) + (ey0-cy)*(ey0-cy)))
	ln := 70.0
	if pxlen > 1 {
		ln = 70.0 / pxlen
	}
	axes := [3]math3d.Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	for i, dir := range axes {
		ep := center.Add(dir.MulScalar(float32(ln)))
		ex, ey, ok2 := e.project(ep)
		if !ok2 {
			continue
		}
		d := math.Sqrt(float64((fx-ex)*(fx-ex) + (fy-ey)*(fy-ey)))
		if d < 12 {
			return i
		}
	}
	return -1
}

// updateDrag gizmo 拖拽（视口内坐标）。
func (e *Editor) updateDrag(mx, my int) {
	if e.sel < 0 || e.sel >= len(e.level.Insts) || e.drag < 0 {
		return
	}
	inst := e.level.Insts[e.sel]
	fx, fy := float32(mx), float32(my)
	cam := e.cam.Camera()
	viewDir := cam.Pos.Sub(e.cam.Target).Normalized()
	switch e.mode {
	case 0: // 移动
		axis := [3]math3d.Vec3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}[e.drag]
		p, ok := e.unproject(fx, fy, inst.Pos, viewDir)
		if ok && e.dragStartOK {
			delta := p.Sub(e.dragAnchor)
			inst.Pos = inst.Pos.Add(axis.MulScalar(delta.Dot(axis)))
			e.dragAnchor = p
		}
		if ok && !e.dragStartOK {
			e.dragAnchor = p
			e.dragStartOK = true
		}
	case 1: // 旋转
		cx, cy, ok := e.project(inst.Pos)
		if ok {
			ang := float32(math.Atan2(float64(fy-cy), float64(fx-cx)))
			if e.dragStartOK {
				da := ang - e.dragAnchorAng
				switch e.drag {
				case 0:
					inst.RotX += da
				case 1:
					inst.RotY += da
				case 2:
					inst.RotZ += da
				}
			}
			e.dragAnchorAng = ang
			e.dragStartOK = true
		}
	case 2: // 缩放
		cx, cy, ok := e.project(inst.Pos)
		if ok {
			dc := math.Sqrt(float64((fx-cx)*(fx-cx) + (fy-cy)*(fy-cy)))
			if e.dragStartOK && e.dragAnchorDist > 1 {
				ratio := float32(dc / e.dragAnchorDist)
				if ratio > 0.01 && ratio < 20 {
					inst.Scale *= ratio
				}
			}
			e.dragAnchorDist = dc
			e.dragStartOK = true
		}
	}
}

// pickAt 视口拾取实例（逐对象拾取渲染）。
func (e *Editor) pickAt(mx, my int) int {
	if mx < 0 || mx >= e.vpW || my < 0 || my >= e.vpH {
		return -1
	}
	pickBuf := make([]byte, e.vpW*e.vpH*4)
	rd := render.NewRenderer(e.vpW, e.vpH)
	for idx, i := range e.level.Insts {
		if !i.Visible || i.IsSprite {
			continue
		}
		ro := i.RenderObj()
		col := render.PickColor(idx + 1)
		ro.ColorOverride = &col
		rd.Clear(pickBuf)
		rd.Render(pickBuf, e.cam.Camera(), []render.Object{ro})
		pix := pickBuf[(my*e.vpW+mx)*4:]
		if pix[2] == col.R && pix[1] == col.G && pix[0] == col.B {
			return idx
		}
	}
	return -1
}
