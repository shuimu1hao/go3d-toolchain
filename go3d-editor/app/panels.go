package app

import (
	"fmt"
	"time"

	"go2dgame/engine"
	"go3d/mesh"
	"go3deditor/ui"
)

// ---------- 面板绘制 ----------

// panelBtn 是工具栏/面板按钮。
type panelBtn struct {
	x, y, w, h int
	label      string
	active     bool // 高亮（模式选中态）
	action     func()
}

var uiColorBg = engine.Color{R: 40, G: 44, B: 52}
var uiColorPanel = engine.Color{R: 52, G: 57, B: 66}
var uiColorPanel2 = engine.Color{R: 46, G: 50, B: 58}
var uiColorBorder = engine.Color{R: 28, G: 31, B: 37}
var uiColorText = engine.Color{R: 220, G: 224, B: 232}
var uiColorDim = engine.Color{R: 140, G: 148, B: 162}
var uiColorAccent = engine.Color{R: 90, G: 150, B: 220}
var uiColorSel = engine.Color{R: 70, G: 110, B: 170}
var uiColorBtn = engine.Color{R: 66, G: 72, B: 84}
var uiColorBtnHover = engine.Color{R: 84, G: 92, B: 108}

// drawToolbar 顶部工具栏：两行（基本体 / 模式+视图+操作）。
func (e *Editor) drawToolbar(c *engine.Canvas) {
	uiBtns = uiBtns[:0] // 每帧清空按钮命中表（Draw 阶段注册，Update 阶段点击查询）
	c.FillRect(0, 0, c.W, ToolbarH, uiColorPanel)
	c.FillRect(0, ToolbarH-1, c.W, 1, uiColorBorder)

	// 第一行：添加基本体
	y := 4
	labels := []struct {
		label string
		t     ObjType
	}{
		{"＋立方体", TCube}, {"＋球体", TSphere}, {"＋圆柱", TCylinder},
		{"＋圆锥", TCone}, {"＋圆环", TTorus}, {"＋平面", TPlane},
	}
	x := 10
	for _, b := range labels {
		w := ui.TextW(b.label) + 16
		drawBtn(c, x, y, w, 24, b.label, false, func() { e.AddObject(b.t) })
		x += w + 6
	}
	// 第一行中段：模型格式导入/导出（建模模式显示）
	if e.editMode == EditModel {
		fmts := []struct {
			label string
			cb    func()
		}{
			{"OBJ↓", func() { e.importOBJ() }},
			{"OBJ↑", func() { e.exportOBJ() }},
			{"STL↓", func() { e.importSTL() }},
			{"GLB↓", func() { e.importGLB() }},
			{"GLB↑", func() { e.exportGLB() }},
		}
		for _, b := range fmts {
			w := ui.TextW(b.label) + 12
			drawBtn(c, x, y, w, 24, b.label, false, b.cb)
			x += w + 6
		}
	}
	// 第一行右端：保存/载入
	sw := ui.TextW("保存") + 14
	drawBtn(c, c.W-PropW-10-sw*2-6, y, sw, 24, "保存", false, func() { e.saveDoc() })
	drawBtn(c, c.W-PropW-10-sw, y, sw, 24, "载入", false, func() { e.loadDoc() })

	// 第二行：编辑模式 + 变换模式 + 视图 + 开关
	y = 34
	x = 10
	modes := []struct {
		label string
		m     EditMode
	}{
		{"建模", EditModel},
		{"草图", EditSketch},
		{"骨骼", EditBone},
		{"动画", EditAnim},
	}
	for _, b := range modes {
		w := ui.TextW(b.label) + 14
		drawBtn(c, x, y, w, 24, b.label, e.editMode == b.m, func() { e.SetEditMode(b.m) })
		x += w + 6
	}
	x += 8
	// 变换模式（建模模式）
	tmodes := []struct {
		label string
		m     TransformMode
	}{
		{"移动[G]", ModeMove},
		{"旋转[R]", ModeRotate},
		{"缩放[S]", ModeScale},
	}
	for _, b := range tmodes {
		w := ui.TextW(b.label) + 14
		drawBtn(c, x, y, w, 24, b.label, e.mode == b.m && e.editMode != EditSketch, func() { e.SetMode(b.m) })
		x += w + 6
	}
	x += 8
	// 模式专属按钮
	switch e.editMode {
	case EditModel:
		// 布尔运算（选中两个对象：当前 = A，上一个选中 = B）
		booleans := []struct {
			label string
			op    CSGOp
		}{
			{"并集", CSGUnion}, {"差集A-B", CSGSubtract}, {"交集", CSGIntersect},
		}
		for _, b := range booleans {
			w := ui.TextW(b.label) + 12
			drawBtn(c, x, y, w, 24, b.label, false, func() { e.csgApply(b.op) })
			x += w + 6
		}
		x += 6
		drawBtn(c, x, y, ui.TextW("导入OBJ")+12, 24, "导入OBJ", false, func() { e.importOBJ() })
		x += ui.TextW("导入OBJ") + 18
		_ = x
	case EditSketch:
		skTools := []struct {
			label string
			t     int
		}{
			{"折线", 0}, {"矩形", 1}, {"圆", 2},
		}
		for _, b := range skTools {
			w := ui.TextW(b.label) + 12
			drawBtn(c, x, y, w, 24, b.label, e.sketchTool == b.t, func() { e.sketchTool = b.t; e.sketchPt0 = Vec2{} })
			x += w + 6
		}
		x += 6
		planes := []struct {
			label string
			p     SketchPlane
		}{
			{"前视", PlaneXY}, {"顶视", PlaneXZ}, {"右视", PlaneYZ},
		}
		for _, b := range planes {
			w := ui.TextW(b.label) + 12
			drawBtn(c, x, y, w, 24, b.label, e.sketch != nil && e.sketch.Plane == b.p, func() { e.sketchSetPlane(b.p) })
			x += w + 6
		}
		x += 6
		drawBtn(c, x, y, ui.TextW("闭合[Enter]")+12, 24, "闭合[Enter]", false, func() { e.sketchClose() })
		x += ui.TextW("闭合[Enter]") + 18
		drawBtn(c, x, y, ui.TextW("拉伸")+12, 24, "拉伸", false, func() { e.sketchExtrude() })
		x += ui.TextW("拉伸") + 18
		drawBtn(c, x, y, ui.TextW("清除")+12, 24, "清除", false, func() { e.sketchClear() })
	case EditBone:
		drawBtn(c, x, y, ui.TextW("＋骨骼")+12, 24, "＋骨骼", false, func() { e.boneAdd() })
		x += ui.TextW("＋骨骼") + 18
		drawBtn(c, x, y, ui.TextW("绑定权重")+12, 24, "绑定权重", false, func() { e.boneBind() })
		x += ui.TextW("绑定权重") + 18
		drawBtn(c, x, y, ui.TextW("重置姿势")+12, 24, "重置姿势", false, func() { e.boneResetPose() })
		x += ui.TextW("重置姿势") + 18
		drawBtn(c, x, y, ui.TextW("删骨骼")+12, 24, "删骨骼", false, func() { e.boneDelete() })
	case EditAnim:
		drawBtn(c, x, y, ui.TextW("＋关键帧")+12, 24, "＋关键帧", false, func() { e.animAddKey() })
		x += ui.TextW("＋关键帧") + 18
		playLabel := "播放"
		if o := e.selObj(); o != nil && o.AnimPlaying {
			playLabel = "暂停"
		}
		drawBtn(c, x, y, ui.TextW(playLabel)+12, 24, playLabel, false, func() { e.animPlay() })
		x += ui.TextW(playLabel) + 18
		drawBtn(c, x, y, ui.TextW("停止")+12, 24, "停止", false, func() { e.animStop() })
		x += ui.TextW("停止") + 18
		drawBtn(c, x, y, ui.TextW("清动画")+12, 24, "清动画", false, func() { e.animClear() })
	}
	// 帮助
	ui.DrawText(c, x+10, y+5, "F1 帮助  Ctrl+S 保存  Ctrl+O 载入", uiColorDim)
	_ = uiColorBg
}

// drawBtn 绘制按钮（含悬停/选中高亮）。
func drawBtn(c *engine.Canvas, x, y, w, h int, label string, active bool, action func()) {
	col := uiColorBtn
	if active {
		col = uiColorAccent
	}
	c.FillRect(x, y, w, h, col)
	c.Rect(x, y, w, h, uiColorBorder)
	ui.DrawText(c, x+8, y+(h-ui.LineH())/2+1, label, uiColorText)
	// action 由 panelClick 的按钮表驱动（绘制阶段只登记）
	registerBtn(x, y, w, h, action)
}

// registerBtn 登记按钮命中区域（全局按钮表，每帧重置）。
var uiBtns []*panelBtn

func registerBtn(x, y, w, h int, action func()) {
	uiBtns = append(uiBtns, &panelBtn{x: x, y: y, w: w, h: h, action: action})
}

// panelClick 处理主画布坐标的按钮/面板点击。
func (e *Editor) panelClick(x, y int) {
	// 工具栏按钮
	for _, b := range uiBtns {
		if x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			b.action()
			return
		}
	}
	// 模型树
	if x < TreeW {
		e.treeClick(x, y)
		return
	}
	// 属性面板
	if x >= e.cW()-PropW {
		e.propClick(x, y)
		return
	}
}

// cW 返回窗口宽度（从引擎画布读）。
func (e *Editor) cW() int {
	if e.eng != nil && e.eng.Canvas() != nil {
		return e.eng.Canvas().W
	}
	return 960
}

// ---------- 模型树 ----------

// treeRowH 树行高。
const treeRowH = 26

// drawTree 左侧模型树（SolidWorks FeatureManager 风格）。
func (e *Editor) drawTree(c *engine.Canvas) {
	c.FillRect(0, ToolbarH, TreeW, c.H-ToolbarH, uiColorPanel2)
	c.FillRect(TreeW-1, ToolbarH, 1, c.H-ToolbarH, uiColorBorder)
	// 标题
	ui.DrawText(c, 10, ToolbarH+8, "模型树", uiColorAccent)
	// 零件根节点
	rootY := ToolbarH + 30
	ui.DrawText(c, 18, rootY, "▸ "+e.doc.Name, uiColorText)
	// 特征列表
	y := rootY + 24
	for i, o := range e.doc.Objs {
		rowY := y + i*treeRowH
		// 选中高亮
		if i == e.sel {
			c.FillRect(2, rowY, TreeW-4, treeRowH, uiColorSel)
		}
		// 可见性开关（小方块）
		visX := 14
		visCol := uiColorDim
		if o.Visible {
			visCol = engine.Color{R: 110, G: 210, B: 130}
		}
		c.FillRect(visX, rowY+7, 12, 12, visCol)
		c.Rect(visX, rowY+7, 12, 12, uiColorBorder)
		// 类型短名
		ui.DrawText(c, 34, rowY+7, o.Type.ShortName(), uiColorDim)
		// 名称（重命名时显示输入框）
		name := o.Name
		if e.renaming && e.renameIdx == i {
			name = e.renameBuf + "|"
		}
		ui.DrawText(c, 34+ui.TextW(o.Type.ShortName())+8, rowY+7, name, uiColorText)
	}
	// 底部提示
	if e.editMode == EditBone {
		ui.DrawText(c, 10, c.H-StatusH-16, "A加骨骼 点击选骨骼 Gizmo编辑", uiColorDim)
	} else {
		ui.DrawText(c, 10, c.H-StatusH-16, "双击重命名  ↑↓排序 Del删除", uiColorDim)
	}

	// 骨骼树（骨骼模式下，显示选中对象的骨骼层级）
	if e.editMode == EditBone {
		o := e.selObj()
		if o != nil && o.Skeleton != nil {
			by := y + len(e.doc.Objs)*treeRowH + 8
			ui.DrawText(c, 10, by, "─ 骨骼树", uiColorAccent)
			by += 22
			for i := range o.Skeleton.Bones {
				b := o.Skeleton.Bones[i]
				rowY := by + i*treeRowH
				if i == e.selBone {
					c.FillRect(2, rowY, TreeW-4, treeRowH, uiColorSel)
				}
				indent := "    "
				if b.Parent >= 0 {
					indent = "  └─"
				}
				ui.DrawText(c, 14, rowY+7, indent+b.Name, uiColorText)
				// 骨骼世界位置简显
				pos := o.Skeleton.BoneWorldPos(i)
				ui.DrawText(c, 120, rowY+7, fmt.Sprintf("(%.1f,%.1f,%.1f)", pos.X, pos.Y, pos.Z), uiColorDim)
			}
		}
	}
}

// drawTimeline 动画模式：底部时间轴（在状态栏上方）。
func (e *Editor) drawTimeline(c *engine.Canvas) {
	if e.editMode != EditAnim {
		return
	}
	o := e.selObj()
	if o == nil || o.Anim == nil {
		return
	}
	ty := e.vpY + e.vpH - 34
	tw := e.vpW - 20
	tx := e.vpX + 10
	// 轨道背景
	c.FillRect(tx, ty, tw, 14, engine.Color{R: 30, G: 34, B: 42})
	c.Rect(tx, ty, tw, 14, uiColorBorder)
	// 进度
	dur := o.Anim.Duration
	if dur <= 0 {
		dur = 1
	}
	px := int(float32(tw) * (o.AnimTime / dur))
	if px > tw {
		px = tw
	}
	c.FillRect(tx, ty, px, 14, uiColorAccent)
	// 刻度（每 0.5s）
	for t := float32(0); t <= dur; t += 0.5 {
		sx := tx + int(float32(tw)*(t/dur))
		c.FillRect(sx, ty+13, 1, 3, uiColorDim)
	}
	ui.DrawText(c, tx, ty-16, fmt.Sprintf("时间轴  t=%.2f/%.2fs  %s", o.AnimTime, dur, map[bool]string{true: "▶播放中", false: "⏸暂停"}[o.AnimPlaying]), uiColorText)
}

// treeClick 模型树点击。
func (e *Editor) treeClick(x, y int) {
	if x >= e.cW()-PropW {
		return
	}
	rootY := ToolbarH + 30 + 24
	idx := (y - rootY) / treeRowH
	if idx < 0 || idx >= len(e.doc.Objs) {
		if y < rootY && e.eng != nil && e.eng.Input() != nil {
			// 点击根节点区域：取消选择
			if !e.eng.Input().MouseLeftPressed {
			}
		}
		e.sel = -1
		e.fieldFocus = -1
		return
	}
	// 可见性小方块
	if x >= 14 && x <= 26 && y >= rootY+idx*treeRowH+7 && y <= rootY+idx*treeRowH+19 {
		e.doc.Objs[idx].Visible = !e.doc.Objs[idx].Visible
		e.sel = idx
		return
	}
	e.sel = idx
	e.fieldFocus = -1
	// 双击重命名（同一行、400ms 内）
	now := time.Now()
	if idx == e.lastClickIdx && now.Sub(e.lastClickTime) < 400*time.Millisecond {
		e.renaming = true
		e.renameBuf = e.doc.Objs[idx].Name
		e.renameIdx = idx
	}
	e.lastClickIdx = idx
	e.lastClickTime = now
}

// ---------- 属性面板 ----------

// propField 属性字段。
type propField struct {
	label string
	value string
}

// drawProps 右侧属性面板。
func (e *Editor) drawProps(c *engine.Canvas) {
	px := c.W - PropW
	c.FillRect(px, ToolbarH, PropW, c.H-ToolbarH, uiColorPanel2)
	c.FillRect(px, ToolbarH, 1, c.H-ToolbarH, uiColorBorder)
	ui.DrawText(c, px+10, ToolbarH+8, "属性", uiColorAccent)

	// 骨骼模式：显示选中骨骼属性
	if e.editMode == EditBone {
		o := e.selObj()
		if o == nil || o.Skeleton == nil || e.selBone < 0 || e.selBone >= len(o.Skeleton.Bones) {
			ui.DrawText(c, px+12, ToolbarH+40, "未选中骨骼", uiColorDim)
			ui.DrawText(c, px+12, ToolbarH+58, "先选中对象并添加骨骼", uiColorDim)
			return
		}
		b := o.Skeleton.Bones[e.selBone]
		y := ToolbarH + 34
		ui.DrawText(c, px+12, y, "骨骼:", uiColorDim)
		ui.DrawText(c, px+80, y, b.Name, uiColorText)
		y += 26
		ui.DrawText(c, px+12, y, "父:", uiColorDim)
		parent := "-"
		if b.Parent >= 0 {
			parent = o.Skeleton.Bones[b.Parent].Name
		}
		ui.DrawText(c, px+80, y, parent, uiColorText)
		y += 32
		labels := []string{"局部X:", "局部Y:", "局部Z:"}
		for i := 0; i < 3; i++ {
			ui.DrawText(c, px+12, y, labels[i], uiColorDim)
			v := []float32{b.Pos.X, b.Pos.Y, b.Pos.Z}[i]
			ui.DrawText(c, px+80, y, fmt.Sprintf("%.3f", v), uiColorText)
			y += 24
		}
		y += 8
		ui.DrawText(c, px+12, y, "旋转°:", uiColorDim)
		ui.DrawText(c, px+80, y, fmt.Sprintf("%.1f %.1f %.1f", b.RotX*180/3.14159, b.RotY*180/3.14159, b.RotZ*180/3.14159), uiColorText)
		y += 34
		drawBtn(c, px+16, y, 100, 26, "＋骨骼[A]", false, func() { e.boneAdd() })
		drawBtn(c, px+124, y, 100, 26, "删骨骼", false, func() { e.boneDelete() })
		y += 36
		drawBtn(c, px+16, y, 100, 26, "绑定权重", false, func() { e.boneBind() })
		drawBtn(c, px+124, y, 100, 26, "重置姿势", false, func() { e.boneResetPose() })
		return
	}
	// 动画模式：显示动画信息
	if e.editMode == EditAnim {
		o := e.selObj()
		if o == nil || o.Anim == nil {
			ui.DrawText(c, px+12, ToolbarH+40, "无动画", uiColorDim)
			ui.DrawText(c, px+12, ToolbarH+58, "选中对象→骨骼模式加骨骼→加关键帧", uiColorDim)
			return
		}
		y := ToolbarH + 34
		ui.DrawText(c, px+12, y, "动画:", uiColorDim)
		ui.DrawText(c, px+80, y, o.Anim.Name, uiColorText)
		y += 26
		ui.DrawText(c, px+12, y, "时长:", uiColorDim)
		ui.DrawText(c, px+80, y, fmt.Sprintf("%.2f s", o.Anim.Duration), uiColorText)
		y += 26
		ui.DrawText(c, px+12, y, "轨道:", uiColorDim)
		ui.DrawText(c, px+80, y, fmt.Sprintf("%d", len(o.Anim.Tracks)), uiColorText)
		y += 26
		ui.DrawText(c, px+12, y, "循环:", uiColorDim)
		loop := "关"
		if o.Anim.Loop {
			loop = "开"
		}
		ui.DrawText(c, px+80, y, loop, uiColorText)
		y += 34
		drawBtn(c, px+16, y, 100, 26, "＋关键帧", false, func() { e.animAddKey() })
		drawBtn(c, px+124, y, 100, 26, "播放/暂停", false, func() { e.animPlay() })
		y += 36
		drawBtn(c, px+16, y, 100, 26, "停止", false, func() { e.animStop() })
		drawBtn(c, px+124, y, 100, 26, "清动画", false, func() { e.animClear() })
		return
	}

	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		ui.DrawText(c, px+12, ToolbarH+40, "未选中特征", uiColorDim)
		ui.DrawText(c, px+12, ToolbarH+58, "在视口或模型树中选择", uiColorDim)
		return
	}
	o := e.doc.Objs[e.sel]
	y := ToolbarH + 34
	// 名称
	ui.DrawText(c, px+12, y, "名称:", uiColorDim)
	name := o.Name
	if e.fieldFocus == 0 {
		name = e.fieldBuf + "|"
	}
	drawPropValue(c, px+80, y, PropW-92, name, e.fieldFocus == 0, 0, func() {})
	// 类型
	ui.DrawText(c, px+12, y+24, "类型:", uiColorDim)
	ui.DrawText(c, px+80, y+24, o.Type.Name(), uiColorText)
	// 位置
	y += 56
	pos := []float32{o.Pos.X, o.Pos.Y, o.Pos.Z}
	labels := []string{"X:", "Y:", "Z:"}
	for i := 0; i < 3; i++ {
		ui.DrawText(c, px+12, y, labels[i], uiColorDim)
		val := fmt.Sprintf("%.3f", pos[i])
		if e.fieldFocus == 1+i {
			val = e.fieldBuf + "|"
		}
		drawPropValue(c, px+40, y, 90, val, e.fieldFocus == 1+i, 1+i, func() {})
		y += 24
	}
	// 旋转（度）
	ui.DrawText(c, px+12, y+2, "旋转°:", uiColorDim)
	rot := []float32{o.RotX * 180 / 3.14159, o.RotY * 180 / 3.14159, o.RotZ * 180 / 3.14159}
	y += 26
	for i := 0; i < 3; i++ {
		ui.DrawText(c, px+12, y, labels[i], uiColorDim)
		val := fmt.Sprintf("%.1f", rot[i])
		if e.fieldFocus == 4+i {
			val = e.fieldBuf + "|"
		}
		drawPropValue(c, px+40, y, 90, val, e.fieldFocus == 4+i, 4+i, func() {})
		y += 24
	}
	// 缩放
	ui.DrawText(c, px+12, y+2, "缩放:", uiColorDim)
	val := fmt.Sprintf("%.3f", o.Scale)
	if e.fieldFocus == 7 {
		val = e.fieldBuf + "|"
	}
	drawPropValue(c, px+80, y+2, 100, val, e.fieldFocus == 7, 7, func() {})
	y += 32

	// 颜色色板
	ui.DrawText(c, px+12, y, "颜色:", uiColorDim)
	y += 22
	palette := []mesh.Color{
		mesh.Col(90, 150, 220), mesh.Col(220, 120, 150), mesh.Col(120, 180, 110),
		mesh.Col(230, 150, 70), mesh.Col(170, 120, 220), mesh.Col(220, 200, 90),
		mesh.Col(200, 90, 90), mesh.Col(150, 150, 160), mesh.Col(255, 255, 255),
	}
	for i, col := range palette {
		cx := px + 16 + (i%6)*26
		cy := y + (i/6)*26
		c.FillRect(cx, cy, 22, 22, engine.Color{R: col.R, G: col.G, B: col.B})
		c.Rect(cx, cy, 22, 22, uiColorBorder)
		if o.Color == col {
			c.Rect(cx-2, cy-2, 26, 26, engine.Color{R: 255, G: 220, B: 80})
		}
	}
	y += 60

	// 操作按钮
	drawBtn(c, px+16, y, 100, 26, "复制[C]", false, func() { e.duplicateSelected() })
	drawBtn(c, px+124, y, 100, 26, "删除[Del]", false, func() { e.deleteSelected() })
	y += 36
	visLabel := "显示"
	if !o.Visible {
		visLabel = "隐藏"
	}
	drawBtn(c, px+16, y, 100, 26, visLabel+"[H]", false, func() { e.toggleVisible() })
	drawBtn(c, px+124, y, 100, 26, "取景[F]", false, func() { e.frameSelected() })
}

// drawPropValue 属性值框。
func drawPropValue(c *engine.Canvas, x, y, w int, val string, focus bool, field int, _ func()) {
	col := uiColorPanel
	if focus {
		col = engine.Color{R: 60, G: 80, B: 110}
	}
	c.FillRect(x, y, w, 20, col)
	c.Rect(x, y, w, 20, uiColorBorder)
	tcol := uiColorText
	if !focus {
		tcol = uiColorText
	}
	ui.DrawText(c, x+5, y+3, val, tcol)
}

// propClick 属性面板点击。
func (e *Editor) propClick(x, y int) {
	// 先处理已登记的按钮
	for _, b := range uiBtns {
		if x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			b.action()
			return
		}
	}
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return
	}
	px := e.cW() - PropW
	o := e.doc.Objs[e.sel]
	// 字段点击区域
	type fieldRect struct {
		x, y, w, h int
		f          int
	}
	fields := []fieldRect{
		{px + 80, ToolbarH + 34, PropW - 92, 20, 0}, // 名称
	}
	y0 := ToolbarH + 34 + 56
	for i := 0; i < 3; i++ {
		fields = append(fields, fieldRect{px + 40, y0 + i*24, 90, 20, 1 + i})
	}
	y1 := y0 + 3*24 + 2 + 26
	for i := 0; i < 3; i++ {
		fields = append(fields, fieldRect{px + 40, y1 + i*24, 90, 20, 4 + i})
	}
	y2 := y1 + 3*24 + 2
	fields = append(fields, fieldRect{px + 80, y2, 100, 20, 7})
	for _, f := range fields {
		if x >= f.x && x < f.x+f.w && y >= f.y && y < f.y+f.h {
			e.fieldFocus = f.f
			e.fieldBuf = e.fieldValue(f.f)
			e.fieldBufEdited = false
			return
		}
	}
	// 色板
	py := y2 + 32 + 22
	for i, col := range propPalette() {
		cx := px + 16 + (i%6)*26
		cy := py + (i/6)*26
		if x >= cx && x < cx+22 && y >= cy && y < cy+22 {
			o.Color = col
			return
		}
	}
	e.fieldFocus = -1
}

// propPalette 返回色板（与绘制一致）。
func propPalette() []mesh.Color {
	return []mesh.Color{
		mesh.Col(90, 150, 220), mesh.Col(220, 120, 150), mesh.Col(120, 180, 110),
		mesh.Col(230, 150, 70), mesh.Col(170, 120, 220), mesh.Col(220, 200, 90),
		mesh.Col(200, 90, 90), mesh.Col(150, 150, 160), mesh.Col(255, 255, 255),
	}
}

// fieldValue 返回字段当前值字符串（供编辑预填）。
func (e *Editor) fieldValue(f int) string {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return ""
	}
	o := e.doc.Objs[e.sel]
	switch f {
	case 0:
		return o.Name
	case 1:
		return fmt.Sprintf("%.3f", o.Pos.X)
	case 2:
		return fmt.Sprintf("%.3f", o.Pos.Y)
	case 3:
		return fmt.Sprintf("%.3f", o.Pos.Z)
	case 4:
		return fmt.Sprintf("%.1f", o.RotX*180/3.14159)
	case 5:
		return fmt.Sprintf("%.1f", o.RotY*180/3.14159)
	case 6:
		return fmt.Sprintf("%.1f", o.RotZ*180/3.14159)
	case 7:
		return fmt.Sprintf("%.3f", o.Scale)
	}
	return ""
}

// ---------- 状态栏 ----------

// drawStatus 底部状态栏。
func (e *Editor) drawStatus(c *engine.Canvas) {
	y := c.H - StatusH
	c.FillRect(0, y, c.W, StatusH, uiColorPanel)
	c.FillRect(0, y, c.W, 1, uiColorBorder)
	// 消息（3 秒后消失，恢复提示）
	msg := e.msg
	if msg == "" || time.Since(e.msgTime) > 3*time.Second {
		msg = "中键拖拽旋转 右键平移 滚轮缩放  F1 帮助"
	}
	ui.DrawText(c, 10, y+8, msg, uiColorText)
	// 右侧：对象数/FPS/坐标
	right := fmt.Sprintf("特征 %d  FPS %d  (%.1f,%.1f,%.1f)", len(e.doc.Objs), e.fps, e.cam.Target.X, e.cam.Target.Y, e.cam.Target.Z)
	ui.DrawTextRight(c, c.W-10, y+8, right, uiColorDim)
	_ = uiColorText
}
