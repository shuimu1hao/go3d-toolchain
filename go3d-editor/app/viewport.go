package app

import (
	"math"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/render"
	"go3deditor/ui"
)

// ViewportCam 是轨道相机（围绕 Target 旋转/缩放，Blender/SolidWorks 风格）。
type ViewportCam struct {
	Target    math3d.Vec3
	Dist      float32
	Yaw       float32
	Pitch     float32
	FOV       float32
	Near, Far float32
	Ortho     bool
	OrthoSize float32
}

// NewViewportCam 返回默认轨道相机（等轴测视角）。
func NewViewportCam() *ViewportCam {
	return &ViewportCam{
		Target:    math3d.Vec3{0, 0.5, 0},
		Dist:      7,
		Yaw:       0.78, // ~45°
		Pitch:     0.52, // ~30°
		FOV:       50 * math.Pi / 180,
		Near:      0.05,
		Far:       500,
		Ortho:     false,
		OrthoSize: 6,
	}
}

// Camera 构造渲染相机（Pos 由轨道参数解出，看向 Target）。
func (v *ViewportCam) Camera() *render.Camera {
	cp := float32(math.Cos(float64(v.Pitch)))
	offset := math3d.Vec3{
		v.Dist * cp * float32(math.Sin(float64(v.Yaw))),
		v.Dist * float32(math.Sin(float64(v.Pitch))),
		v.Dist * cp * float32(math.Cos(float64(v.Yaw))),
	}
	target := v.Target
	return &render.Camera{
		Pos:        v.Target.Add(offset),
		FOV:        v.FOV,
		Near:       v.Near,
		Far:        v.Far,
		LookTarget: &target,
		Ortho:      v.Ortho,
		OrthoSize:  v.OrthoSize,
	}
}

// SetView 切换到标准视图：0=顶视 1=前视 2=右视 3=等轴测。
func (v *ViewportCam) SetView(mode int) {
	v.Pitch = 0.52
	v.Yaw = 0.78
	switch mode {
	case 0: // 顶视：从 +Y 看向下
		v.Yaw = 0
		v.Pitch = math.Pi/2 - 0.01
	case 1: // 前视：从 +Z 看向 -Z（XY 平面）
		v.Yaw = 0
		v.Pitch = 0
	case 2: // 右视：从 +X 看向 -X
		v.Yaw = math.Pi / 2
		v.Pitch = 0
	case 3: // 等轴测
		v.Yaw = math.Pi / 4
		v.Pitch = 0.6155
	}
}

// Zoom 缩放：factor >1 拉近，<1 拉远。
func (v *ViewportCam) Zoom(factor float32) {
	v.Dist /= factor
	if v.Dist < 0.3 {
		v.Dist = 0.3
	}
	if v.Dist > 300 {
		v.Dist = 300
	}
}

// Orbit 轨道旋转。
func (v *ViewportCam) Orbit(dx, dy float32) {
	v.Yaw += dx * 0.008
	v.Pitch -= dy * 0.008
	if v.Pitch > 1.55 {
		v.Pitch = 1.55
	}
	if v.Pitch < -1.55 {
		v.Pitch = -1.55
	}
}

// Pan 平移目标（世界坐标增量，相对相机方向）。
func (v *ViewportCam) Pan(dx, dy float32) {
	// 屏幕 dx/dy → 世界：右向量与上向量
	cp := float32(math.Cos(float64(v.Pitch)))
	fwd := math3d.Vec3{cp * float32(math.Sin(float64(v.Yaw))), 0, cp * float32(math.Cos(float64(v.Yaw)))} // 水平前向
	right := math3d.Vec3{fwd.Z, 0, -fwd.X}
	up := math3d.Vec3{0, 1, 0}
	scale := v.Dist * 0.0016
	v.Target = v.Target.Sub(right.MulScalar(dx * scale)).Add(up.MulScalar(dy * scale))
}

// ---------- 视口投影工具 ----------

// viewProj 返回视图投影矩阵（供 2D 叠加绘制/反投影使用）。
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

// project 把世界点投影到视口屏幕坐标（返回视口内坐标；ok=false 表示在相机后方）。
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

// unproject 把视口屏幕坐标反投影到「过 anchor、法线为 viewDir 的平面」上的世界点。
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

// ---------- 视口渲染 ----------

// drawViewport 渲染视口到主画布（含网格/轴/gizmo 叠加）。
func (e *Editor) drawViewport(c *engine.Canvas) {
	// 背景
	e.rd.Clear(e.vpPixels)
	// 物体
	objs := e.renderObjs()
	e.rd.Render(e.vpPixels, e.cam.Camera(), objs)
	// 选中物体线框高亮
	if e.sel >= 0 && e.sel < len(e.doc.Objs) && e.doc.Objs[e.sel].Visible {
		yellow := meshCol(255, 220, 80)
		o := e.doc.Objs[e.sel].RenderObj()
		o.WireColor = &yellow
		e.rd.Wireframe = true
		e.rd.Render(e.vpPixels, e.cam.Camera(), []render.Object{o})
		e.rd.Wireframe = false
	}
	// 上屏到视口区域
	e.blitViewport(c)
	// 2D 叠加：网格线、坐标轴、gizmo
	e.drawGridOverlay(c)
	e.drawAxisIndicator(c)
	if e.editMode == EditSketch {
		e.drawSketchOverlay(c)
	}
	if e.editMode == EditBone {
		e.drawBones(c)
	}
	e.drawGizmo(c)
}

// drawSketchOverlay 绘制草图元素（点 + 线段）。
func (e *Editor) drawSketchOverlay(c *engine.Canvas) {
	if e.sketch == nil {
		return
	}
	pts := e.sketch.Points
	if len(pts) == 0 {
		return
	}
	col := engine.Color{R: 255, G: 200, B: 80}
	clip := func(sx, sy float32) (int, int, bool) {
		px, py := int(sx)+e.vpX, int(sy)+e.vpY
		if px < e.vpX || px >= e.vpX+e.vpW || py < e.vpY || py >= e.vpY+e.vpH {
			return 0, 0, false
		}
		return px, py, true
	}
	var prevX, prevY int
	havePrev := false
	for i, p := range pts {
		w := e.sketch.LocalToWorld(p)
		sx, sy, ok := e.project(w)
		if !ok {
			havePrev = false
			continue
		}
		px, py, ok2 := clip(sx, sy)
		if ok2 {
			c.FillRect(px-2, py-2, 5, 5, col)
			if havePrev {
				c.Line(prevX, prevY, px, py, col)
			}
			prevX, prevY = px, py
			havePrev = true
		}
		_ = i
	}
	// 闭合：连线首尾
	if e.sketch.Closed && len(pts) >= 3 {
		w0 := e.sketch.LocalToWorld(pts[0])
		sx0, sy0, ok0 := e.project(w0)
		if ok0 && havePrev {
			px0, py0, ok3 := clip(sx0, sy0)
			if ok3 {
				c.Line(prevX, prevY, px0, py0, col)
			}
		}
	}
	// 矩形/圆的第二点提示
	if e.sketchHasPt0 {
		p0 := e.sketch.LocalToWorld(e.sketchPt0)
		sx0, sy0, ok0 := e.project(p0)
		if ok0 {
			px0, py0, _ := clip(sx0, sy0)
			c.FillRect(px0-3, py0-3, 7, 7, engine.Color{R: 120, G: 220, B: 120})
			// 实时预览（CAD 风格）：矩形框 / 圆轮廓跟随鼠标
			if e.eng != nil && e.eng.Input() != nil {
				in := e.eng.Input()
				mx, my := in.MouseX-e.vpX, in.MouseY-e.vpY
				if mx >= 0 && my >= 0 && mx < e.vpW && my < e.vpH {
					wp, ok2 := e.unproject(float32(mx), float32(my), math3d.Vec3{0, 0, 0}, e.sketch.Normal())
					if ok2 {
						cur := e.sketch.WorldToLocal(wp)
						preview := engine.Color{R: 255, G: 160, B: 90}
						switch e.sketchTool {
						case 1: // 矩形预览
							pr := RectSketch(e.sketchPt0.U, e.sketchPt0.V, cur.U, cur.V)
							drawSketchPoly(c, e, clip, pr, preview)
						case 2: // 圆预览（半径=圆心到鼠标）
							dx := cur.U - e.sketchPt0.U
							dy := cur.V - e.sketchPt0.V
							r := float32(mathSqrt(float64(dx*dx + dy*dy)))
							pr := CircleSketch(e.sketchPt0.U, e.sketchPt0.V, r, 48)
							drawSketchPoly(c, e, clip, pr, preview)
						}
					}
				}
			}
		}
	}
}

// drawSketchPoly 投影绘制草图多边形轮廓（预览用）。
func drawSketchPoly(c *engine.Canvas, e *Editor, clip func(float32, float32) (int, int, bool), pts []Vec2, col engine.Color) {
	if len(pts) < 2 {
		return
	}
	var prevX, prevY int
	havePrev := false
	for _, p := range pts {
		w := e.sketch.LocalToWorld(p)
		sx, sy, ok := e.project(w)
		if !ok {
			havePrev = false
			continue
		}
		px, py, ok2 := clip(sx, sy)
		if ok2 {
			if havePrev {
				c.Line(prevX, prevY, px, py, col)
			}
			prevX, prevY = px, py
			havePrev = true
		}
	}
	if havePrev && len(pts) >= 3 {
		w0 := e.sketch.LocalToWorld(pts[0])
		sx0, sy0, ok0 := e.project(w0)
		if ok0 {
			px0, py0, ok3 := clip(sx0, sy0)
			if ok3 {
				c.Line(prevX, prevY, px0, py0, col)
			}
		}
	}
}

// drawBones 绘制骨骼可视化（绿色骨线 + 关节点）。
func (e *Editor) drawBones(c *engine.Canvas) {
	o := e.selObj()
	if o == nil || o.Skeleton == nil {
		return
	}
	s := o.Skeleton
	lineCol := engine.Color{R: 100, G: 220, B: 120}
	jointCol := engine.Color{R: 255, G: 220, B: 120}
	selCol := engine.Color{R: 255, G: 120, B: 80}
	clip := func(sx, sy float32) (int, int, bool) {
		px, py := int(sx)+e.vpX, int(sy)+e.vpY
		if px < e.vpX || px >= e.vpX+e.vpW || py < e.vpY || py >= e.vpY+e.vpH {
			return 0, 0, false
		}
		return px, py, true
	}
	for i := range s.Bones {
		bp := s.BoneWorldPos(i)
		sx, sy, ok := e.project(bp)
		if !ok {
			continue
		}
		px, py, _ := clip(sx, sy)
		col := jointCol
		if i == e.selBone {
			col = selCol
		}
		c.FillRect(px-3, py-3, 7, 7, col)
		// 骨线到父
		if s.Bones[i].Parent >= 0 {
			pp := s.BoneWorldPos(s.Bones[i].Parent)
			px2, py2, ok2 := e.project(pp)
			if ok2 {
				qx, qy, ok3 := clip(px2, py2)
				if ok3 {
					c.Line(px, py, qx, qy, lineCol)
				}
			}
		}
	}
}

// blitViewport 把视口缓冲拷到主画布。
func (e *Editor) blitViewport(c *engine.Canvas) {
	for y := 0; y < e.vpH; y++ {
		dy := e.vpY + y
		if dy < 0 || dy >= c.H {
			continue
		}
		srcOff := y * e.vpW * 4
		dstOff := (dy*c.W + e.vpX) * 4
		copy(c.Pixels[dstOff:dstOff+e.vpW*4], e.vpPixels[srcOff:srcOff+e.vpW*4])
	}
}

// drawGridOverlay 在视口上画 XZ 平面网格线（2D 投影线）。
func (e *Editor) drawGridOverlay(c *engine.Canvas) {
	if !e.showGrid {
		return
	}
	gridCol := engine.Color{R: 70, G: 80, B: 95}
	axisCol := engine.Color{R: 110, G: 120, B: 140}
	half := 5
	// 网格线投影到屏幕，裁剪到视口
	clip := func(sx, sy float32) (int, int, bool) {
		px, py := int(sx)+e.vpX, int(sy)+e.vpY
		if px < e.vpX || px >= e.vpX+e.vpW || py < e.vpY || py >= e.vpY+e.vpH {
			return 0, 0, false
		}
		return px, py, true
	}
	drawLine := func(a, b math3d.Vec3, col engine.Color) {
		ax, ay, ok1 := e.project(a)
		bx, by, ok2 := e.project(b)
		if !ok1 || !ok2 {
			return
		}
		px0, py0, c0 := clip(ax, ay)
		px1, py1, c1 := clip(bx, by)
		if !c0 || !c1 {
			return
		}
		c.Line(px0, py0, px1, py1, col)
	}
	for i := -half; i <= half; i++ {
		col := gridCol
		if i == 0 {
			col = axisCol
		}
		drawLine(math3d.Vec3{float32(i), 0, float32(-half)}, math3d.Vec3{float32(i), 0, float32(half)}, col)
		drawLine(math3d.Vec3{float32(-half), 0, float32(i)}, math3d.Vec3{float32(half), 0, float32(i)}, col)
	}
}

// drawAxisIndicator 视口左下角画坐标轴指示器（X 红 Y 绿 Z 蓝，带字母标签）。
// 方向由相机视图矩阵把世界轴变换到相机空间得到（不依赖世界原点投影，
// 即使原点在相机后方/屏幕外也始终正确）；锚点固定在视口左下角。
func (e *Editor) drawAxisIndicator(c *engine.Canvas) {
	const axisLen = 46 // 轴长（px）
	// 锚点：视口左下角（避开状态栏）
	ax, ay := e.vpX+60, e.vpY+e.vpH-52
	cam := e.cam.Camera()
	var view math3d.Mat4
	if cam.LookTarget != nil {
		view = math3d.LookAt(cam.Pos, *cam.LookTarget, math3d.Vec3{0, 1, 0})
	} else {
		view = math3d.FPSView(cam.Pos, cam.Yaw, cam.Pitch)
	}
	axes := []struct {
		dir   math3d.Vec3
		col   engine.Color
		label string
	}{
		{math3d.Vec3{1, 0, 0}, engine.Color{R: 230, G: 80, B: 80}, "X"},
		{math3d.Vec3{0, 1, 0}, engine.Color{R: 80, G: 210, B: 95}, "Y"},
		{math3d.Vec3{0, 0, 1}, engine.Color{R: 90, G: 140, B: 250}, "Z"},
	}
	// 背景小方块（便于识别）
	c.FillRect(ax-42, ay-34, 100, 62, engine.Color{R: 26, G: 30, B: 38})
	c.Rect(ax-42, ay-34, 100, 62, uiColorBorder)
	for _, a := range axes {
		// 世界轴方向 → 相机空间（Mat3 去掉平移），屏幕 y 向下取反
		cd := view.Mat3().Transform(a.dir)
		dx, dy := float64(cd.X), float64(-cd.Y)
		l := math.Sqrt(dx*dx + dy*dy)
		if l < 1e-9 {
			// 轴与视线平行（顶视时 Y 轴）：画空心点表示朝向屏幕
			c.Circle(ax, ay, 4, a.col)
			ui.DrawText(c, ax+8, ay-8, a.label, a.col)
			continue
		}
		ux, uy := dx/l, dy/l
		ex := ax + int(ux*axisLen)
		ey := ay + int(uy*axisLen)
		// 先画暗影线再画亮线，突出主方向
		c.Line(ax+1, ay+1, ex+1, ey+1, engine.Color{R: 20, G: 22, B: 28})
		c.Line(ax, ay, ex, ey, a.col)
		// 末端小方块 + 字母标签
		c.FillRect(ex-4, ey-4, 8, 8, a.col)
		ui.DrawText(c, ex+6, ey-6, a.label, a.col)
	}
	ui.DrawText(c, ax-34, ay+18, "世界", uiColorDim)
}
