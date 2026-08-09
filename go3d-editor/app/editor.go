package app

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/render"
)

// 布局常量（SolidWorks 风格：顶工具栏/左模型树/右属性/底状态栏/中视口）。
const (
	ToolbarH = 88 // 三行工具栏（SolidWorks 风格：基本体 / 模式 / 全局操作）
	TreeW    = 240
	PropW    = 280
	StatusH  = 32
)

// TransformMode 变换模式。
type TransformMode int

const (
	ModeMove TransformMode = iota
	ModeRotate
	ModeScale
)

// EditMode 编辑模式（建模大类/草图/骨骼/动画）。
type EditMode int

const (
	EditModel EditMode = iota // 建模（基本体+变换+布尔）
	EditSketch                // 草图绘制/拉伸
	EditBone                  // 骨骼绑定
	EditAnim                  // 白模动画
)

// Editor 是编辑器主状态。
type Editor struct {
	eng *engine.Engine
	doc *Document
	cam *ViewportCam
	rd  *render.Renderer

	vpX, vpY, vpW, vpH int
	vpPixels           []byte
	pickPixels         []byte

	sel       int
	selPrev   int // 上次选中（布尔运算 B 对象；CAD 风格：先选 B 再选 A）
	mode      TransformMode
	editMode  EditMode
	showGrid  bool
	autoFrame bool

	// 草图状态
	sketch     *Sketch
	sketchTool int  // 0=折线 1=矩形 2=圆
	sketchPt0  Vec2
	sketchHasPt0 bool // 矩形/圆第一点是否已设置（独立标志，避免 (0,0) 哨兵失效）

	// 骨骼/动画状态
	selBone int

	// gizmo 拖拽状态
	drag          int
	dragStartOK   bool
	dragAnchor    math3d.Vec3
	dragAnchorAng float32
	dragAnchorDist float64

	// 视口导航状态
	orbitDrag bool
	panDrag   bool
	lastMX    int
	lastMY    int

	// 树双击检测
	lastClickIdx   int
	lastClickTime  time.Time

	// UI 编辑状态
	renaming  bool
	renameBuf string
	renameIdx int
	fieldFocus int
	fieldBuf   string
	fieldBufEdited bool

	// 保存对话框
	saveDialogOpen bool
	saveBuf        string
	saveBufEdited  bool
	lastSaveRel    string

	// 吸附（CAD OSNAP 风格：F 开关 + 类型多选）
	snap          bool
	snapMask      int // 位标志：SnapGrid|SnapVertex|SnapMid|SnapCenter
	snapStep      float32
	snapPanelOpen bool

	// 主题（0=暗色 1=白色）与插件面板
	theme           int
	pluginPanelOpen bool

	msg     string
	msgTime time.Time

	// 统计
	frames int
	fps    int
	fpsCnt int
	fpsT   time.Time
}

// New 创建编辑器（w/h 为主画布尺寸）。
func New(w, h int) *Editor {
	e := &Editor{
		doc:      NewDocument("零件1"),
		cam:      NewViewportCam(),
		sel:      -1,
		selPrev:  -1,
		mode:     ModeMove,
		showGrid: true,
		snap:     true,
		snapStep: 0.5,
		snapMask: SnapGrid | SnapVertex,
		lastSaveRel: "hermes11/go3d-editor/scene.json",
	}
	e.theme = themeIndex(loadThemeName())
	if e.theme == 1 {
		SetTheme(ThemeLight)
	}
	e.layout(w, h)
	e.rd = render.NewRenderer(e.vpW, e.vpH)
	if e.theme == 1 {
		e.rd.ClearColor = themeBgMesh(ThemeLight)
	}
	e.vpPixels = make([]byte, e.vpW*e.vpH*4)
	e.pickPixels = make([]byte, e.vpW*e.vpH*4)
	e.fpsT = time.Now()
	return e
}

// layout 计算面板布局。
func (e *Editor) layout(w, h int) {
	e.vpX = TreeW
	e.vpY = ToolbarH
	e.vpW = w - TreeW - PropW
	e.vpH = h - ToolbarH - StatusH
	if e.vpW < 100 {
		e.vpW = 100
	}
	if e.vpH < 100 {
		e.vpH = 100
	}
}

// SetEngine 绑定引擎（绘制需要画布）。
func (e *Editor) SetEngine(eng *engine.Engine) { e.eng = eng }

// Document 返回文档。
func (e *Editor) Document() *Document { return e.doc }

// Selected 返回选中对象（-1 无）。
func (e *Editor) Selected() int { return e.sel }

// SetMode 设置变换模式。
func (e *Editor) SetMode(m TransformMode) { e.mode = m; e.drag = DragNone }

// SetEditMode 切换编辑模式（建模/草图/骨骼/动画）。
func (e *Editor) SetEditMode(m EditMode) {
	e.editMode = m
	e.drag = DragNone
	e.fieldFocus = -1
	e.renaming = false
	switch m {
	case EditSketch:
		if e.sketch == nil {
			e.sketch = NewSketch(PlaneXY)
		}
	case EditBone:
		e.selBone = -1
	case EditAnim:
		e.selBone = -1
	}
}

// SetMessage 显示状态栏消息。
func (e *Editor) SetMessage(format string, args ...any) {
	e.msg = fmt.Sprintf(format, args...)
	e.msgTime = time.Now()
}

// AddObject 添加基本体并选中。
func (e *Editor) AddObject(t ObjType) {
	o := e.doc.Add(t)
	e.sel = len(e.doc.Objs) - 1
	e.autoFrame = false
	e.SetMessage("添加特征: %s", o.Name)
}

// renderObjs 生成渲染对象列表（可见对象）。
func (e *Editor) renderObjs() []render.Object {
	vis := e.doc.VisibleObjs()
	objs := make([]render.Object, 0, len(vis))
	for _, o := range vis {
		objs = append(objs, o.RenderObj())
	}
	return objs
}

// renderPicks 生成拾取对象列表（按原索引顺序，隐藏对象跳过）。
// 返回与 objs 的索引映射（pick 渲染的 i+1 → doc 索引）。
func (e *Editor) renderPicks() []render.Object {
	objs := make([]render.Object, 0, len(e.doc.Objs))
	for _, o := range e.doc.Objs {
		objs = append(objs, o.RenderObj())
	}
	return objs
}

// pickAt 拾取视口 (mx,my) 位置的对象，返回 doc 索引（-1 无）。
func (e *Editor) pickAt(mx, my int) int {
	if mx < 0 || my < 0 || mx >= e.vpW || my >= e.vpH {
		return -1
	}
	e.rd.RenderPick(e.pickPixels, e.cam.Camera(), e.renderPicks())
	i := (my*e.vpW + mx) * 4
	id := render.PickIDFromColor(e.pickPixels[i+2], e.pickPixels[i+1], e.pickPixels[i])
	if id <= 0 || id > len(e.doc.Objs) {
		return -1
	}
	return id - 1
}

// Update 每帧更新（输入分发）。dt 秒。
func (e *Editor) Update(dt float64) {
	if e.eng == nil {
		return
	}
	in := e.eng.Input()
	// 推进所有对象动画
	for _, o := range e.doc.Objs {
		o.UpdateAnim(dt)
	}
	e.handleKeyboard(in)
	e.handleMouse(in)
	// 视口导航（orbit/pan/zoom）
	e.handleNav(in)
	// FPS
	e.frames++
	e.fpsCnt++
	if time.Since(e.fpsT) >= time.Second {
		e.fps = e.fpsCnt
		e.fpsCnt = 0
		e.fpsT = time.Now()
	}
}

// Draw 绘制整帧。
func (e *Editor) Draw(c *engine.Canvas) {
	e.drawViewport(c)
	e.drawToolbar(c)
	e.drawTree(c)
	e.drawProps(c)
	e.drawTimeline(c)
	e.drawStatus(c)
	// 浮动面板（模态，最后绘制在最上层）
	if e.saveDialogOpen {
		e.drawSaveDialog(c)
	}
	if e.snapPanelOpen {
		e.drawSnapPanel(c)
	}
	if e.pluginPanelOpen {
		e.drawPluginPanel(c)
	}
}

// handleKeyboard 处理键盘输入（命令 + 文本编辑）。
func (e *Editor) handleKeyboard(in *engine.Input) {
	// 文本编辑优先（重命名/字段输入/保存对话框）
	if e.renaming || e.fieldFocus >= 0 || e.saveDialogOpen {
		e.handleTextInput(in)
		return
	}
	// 快捷键
	// ---------- 组合键（高级操作：Ctrl/Alt + 字母） ----------
	// Ctrl+字母：X11 keysym 为控制字符 0x01-0x1a
	if in.Pressed(engine.KeyChar('\x13')) { // Ctrl+S 保存（询问文件名/位置）
		e.saveDoc()
		return
	}
	if in.Pressed(engine.KeyChar('\x0f')) { // Ctrl+O 载入
		e.loadDoc()
		return
	}
	if in.Pressed(engine.KeyChar('\x0e')) { // Ctrl+N 新建文档
		e.newDoc()
		return
	}
	if in.Pressed(engine.KeyChar('\x05')) { // Ctrl+E 导出选中 OBJ
		e.exportOBJ()
		return
	}
	if in.Pressed(engine.KeyChar('\x09')) { // Ctrl+I 导入 OBJ
		e.importOBJ()
		return
	}
	if in.Pressed(engine.KeyChar('\x04')) { // Ctrl+D 复制（与单键 C 等效）
		e.duplicateSelected()
		return
	}
	if in.Pressed(engine.KeyChar('\x01')) { // Ctrl+A 不处理
		return
	}
	// Alt+字母：keysym 不变，靠 Alt 修饰键区分
	if in.Down(engine.KeyAltL) || in.Down(engine.KeyAltR) {
		if in.Pressed(engine.KeyChar('f')) { // Alt+F 吸附类型面板
			e.saveDialogOpen = false
			e.pluginPanelOpen = false
			e.snapPanelOpen = !e.snapPanelOpen
			return
		}
	if in.Pressed(engine.KeyEscape) {
		e.sel = -1
		e.drag = DragNone
		return
	}
	if in.Pressed(engine.KeyChar('g')) || in.Pressed(engine.KeyChar('G')) {
		e.SetMode(ModeMove)
		e.SetMessage("模式: 移动 (Gizmo)")
		return
	}
	if in.Pressed(engine.KeyChar('r')) || in.Pressed(engine.KeyChar('R')) {
		e.SetMode(ModeRotate)
		e.SetMessage("模式: 旋转")
		return
	}
	if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyChar('S')) {
		e.SetMode(ModeScale)
		e.SetMessage("模式: 缩放")
		return
	}
	if in.Pressed(engine.KeyDelete) {
		e.deleteSelected()
		return
	}
	if in.Pressed(engine.KeyChar('c')) || in.Pressed(engine.KeyChar('C')) {
		e.duplicateSelected()
		return
	}
	if in.Pressed(engine.KeyChar('h')) || in.Pressed(engine.KeyChar('H')) {
		e.toggleVisible()
		return
	}
	// 视图快捷键
	if in.Pressed(engine.KeyChar('1')) {
		e.cam.SetView(1)
		e.SetMessage("视图: 前视")
		return
	}
	if in.Pressed(engine.KeyChar('2')) {
		e.cam.SetView(3)
		e.SetMessage("视图: 等轴测")
		return
	}
	if in.Pressed(engine.KeyChar('3')) {
		e.cam.SetView(2)
		e.SetMessage("视图: 右视")
		return
	}
	if in.Pressed(engine.KeyChar('7')) {
		e.cam.SetView(0)
		e.SetMessage("视图: 顶视")
		return
	}
	if in.Pressed(engine.KeyChar('p')) || in.Pressed(engine.KeyChar('P')) {
		e.cam.Ortho = !e.cam.Ortho
		if e.cam.Ortho {
			e.SetMessage("投影: 正交")
		} else {
			e.SetMessage("投影: 透视")
		}
		return
	}
	if in.Pressed(engine.KeyChar('F')) { // Shift+F 取景（F 已用于吸附开关）
		e.frameSelected()
		return
	}
		if in.Pressed(engine.KeyChar('t')) { // Alt+T 主题切换
			e.toggleTheme()
			return
		}
		if in.Pressed(engine.KeyChar('b')) { // Alt+B 添加骨骼
			e.boneAdd()
			return
		}
		if in.Pressed(engine.KeyChar('w')) { // Alt+W 绑定权重
			e.boneBind()
			return
		}
		if in.Pressed(engine.KeyChar('k')) { // Alt+K 添加关键帧
			e.animAddKey()
			return
		}
		if in.Pressed(engine.KeyChar('x')) { // Alt+X 清除草图
			e.sketchClear()
			return
		}
		if in.Pressed(engine.KeyChar('1')) { // Alt+1 草图平面：前视
			e.sketchSetPlane(PlaneXY)
			return
		}
		if in.Pressed(engine.KeyChar('2')) { // Alt+2 草图平面：顶视
			e.sketchSetPlane(PlaneXZ)
			return
		}
		if in.Pressed(engine.KeyChar('3')) { // Alt+3 草图平面：右视
			e.sketchSetPlane(PlaneYZ)
			return
		}
	}
	if in.Pressed(engine.KeyHelp) {
		e.showHelp()
		return
	}
	if in.Pressed(engine.KeyChar(' ')) {
		e.showGrid = !e.showGrid
		return
	}
	// F 键：吸附开关（CAD OSNAP 风格）
	if in.Pressed(engine.KeyChar('f')) {
		e.snap = !e.snap
		if e.snap {
			e.SetMessage("吸附: 开（类型见 类型 面板）")
		} else {
			e.SetMessage("吸附: 关 [F]")
		}
		return
	}
	// T 键：切换主题（暗色/白色）
	if in.Pressed(engine.KeyChar('t')) || in.Pressed(engine.KeyChar('T')) {
		e.toggleTheme()
		return
	}
	// Tab 切换编辑模式（建模→草图→骨骼→动画）
	if in.Pressed(engine.KeyTab) {
		next := (e.editMode + 1) % 4
		e.SetEditMode(next)
		names := []string{"建模", "草图", "骨骼", "动画"}
		e.SetMessage("编辑模式: %s", names[next])
		return
	}
	// 草图模式：Enter 闭合
	if e.editMode == EditSketch && (in.Pressed(engine.KeyReturn) || in.Pressed(engine.KeyEnter)) {
		e.sketchClose()
		return
	}
	// 骨骼模式：A 添加骨骼
	if e.editMode == EditBone && (in.Pressed(engine.KeyChar('a')) || in.Pressed(engine.KeyChar('A'))) {
		e.boneAdd()
		return
	}
	// 动画模式：左右方向键推进时间
	if e.editMode == EditAnim {
		o := e.selObj()
		if o != nil && o.Anim != nil {
			step := float32(0.25)
			if in.Pressed(engine.KeyRight) || in.Pressed(engine.KeyRightDup) {
				o.AnimTime += step
				if o.AnimTime > o.Anim.Duration {
					o.AnimTime = o.Anim.Duration
				}
				o.Anim.ApplyToSkeleton(o.Skeleton, o.AnimTime)
				e.SetMessage("t=%.2f", o.AnimTime)
				return
			}
			if in.Pressed(engine.KeyLeft) || in.Pressed(engine.KeyLeftDup) {
				o.AnimTime -= step
				if o.AnimTime < 0 {
					o.AnimTime = 0
				}
				o.Anim.ApplyToSkeleton(o.Skeleton, o.AnimTime)
				e.SetMessage("t=%.2f", o.AnimTime)
				return
			}
		}
	}
}

// showHelp 打开帮助文档：新终端会话（xfce4-terminal），分模块详细说明。
func (e *Editor) showHelp() {
	e.SetMessage("打开帮助会话…（F1 / 帮助按钮）")
	home, err := os.UserHomeDir()
	if err != nil {
		e.SetMessage("无法定位 home 目录")
		return
	}
	helpPath := home + "/hermes11/go3d-editor/HELP.txt"
	// fire-and-forget：不阻塞编辑器
	cmd := exec.Command("bash", "-c",
		"DISPLAY=:0 xfce4-terminal --hold --title=go3d-editor帮助 --command='cat "+helpPath+"' >/dev/null 2>&1")
	if err := cmd.Start(); err != nil {
		e.SetMessage("打开帮助失败: %v", err)
	}
}

// handleTextInput 处理重命名/数值输入/保存对话框输入。
func (e *Editor) handleTextInput(in *engine.Input) {
	if in.Pressed(engine.KeyEscape) {
		e.renaming = false
		e.fieldFocus = -1
		e.saveDialogOpen = false
		return
	}
	if in.Pressed(engine.KeyReturn) || in.Pressed(engine.KeyEnter) {
		e.commitTextInput()
		return
	}
	if in.Pressed(engine.KeyBack) {
		if len(e.buf()) > 0 {
			if e.renaming {
				e.renameBuf = e.renameBuf[:len(e.renameBuf)-1]
			} else if e.saveDialogOpen {
				e.saveBuf = e.saveBuf[:len(e.saveBuf)-1]
			} else {
				e.fieldBuf = e.fieldBuf[:len(e.fieldBuf)-1]
			}
		}
		return
	}
	// 可输入字符：字母数字 负号 小数点 斜杠（路径分隔符）
	inputChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._/ "
	for i := 0; i < len(inputChars); i++ {
		if in.Pressed(engine.KeyChar(inputChars[i])) {
			if e.renaming {
				if len(e.renameBuf) < 24 {
					e.renameBuf += string(inputChars[i])
				}
			} else if e.saveDialogOpen {
				// 首次输入替换预填路径（另存为风格）
				if !e.saveBufEdited {
					e.saveBuf = ""
					e.saveBufEdited = true
				}
				if len(e.saveBuf) < 96 {
					e.saveBuf += string(inputChars[i])
				}
			} else {
				// 首次输入替换预填值（Blender 风格）
				if !e.fieldBufEdited {
					e.fieldBuf = ""
					e.fieldBufEdited = true
				}
				if len(e.fieldBuf) < 16 {
					e.fieldBuf += string(inputChars[i])
				}
			}
			return
		}
	}
}

// buf 返回当前编辑缓冲。
func (e *Editor) buf() string {
	if e.renaming {
		return e.renameBuf
	}
	return e.fieldBuf
}

// commitTextInput 提交文本输入。
func (e *Editor) commitTextInput() {
	if e.saveDialogOpen {
		e.doSave()
		return
	}
	if e.renaming {
		if e.renameIdx >= 0 && e.renameIdx < len(e.doc.Objs) && e.renameBuf != "" {
			e.doc.Objs[e.renameIdx].Name = e.renameBuf
		}
		e.renaming = false
		return
	}
	if e.fieldFocus >= 0 && e.sel >= 0 && e.sel < len(e.doc.Objs) {
		o := e.doc.Objs[e.sel]
		v := parseFloat(e.fieldBuf)
		switch e.fieldFocus {
		case 0:
			if e.fieldBuf != "" {
				o.Name = e.fieldBuf
			}
		case 1:
			o.Pos.X = v
		case 2:
			o.Pos.Y = v
		case 3:
			o.Pos.Z = v
		case 4:
			o.RotX = v * 3.14159265 / 180
		case 5:
			o.RotY = v * 3.14159265 / 180
		case 6:
			o.RotZ = v * 3.14159265 / 180
		case 7:
			o.Scale = v
		}
	}
	e.fieldFocus = -1
	e.fieldBufEdited = false
}

// parseFloat 解析浮点数（空/非法返回 0）。
func parseFloat(s string) float32 {
	var v float32
	var neg bool
	var frac float32 = 0.1
	seenDot := false
	seenDigit := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '-' && i == 0 {
			neg = true
			continue
		}
		if ch == '.' && !seenDot {
			seenDot = true
			continue
		}
		if ch >= '0' && ch <= '9' {
			seenDigit = true
			d := float32(ch - '0')
			if seenDot {
				v += d * frac
				frac *= 0.1
			} else {
				v = v*10 + d
			}
		}
	}
	if !seenDigit {
		return 0
	}
	if neg {
		return -v
	}
	return v
}

// handleMouse 处理视口鼠标（选择/gizmo 拖拽）与面板点击。
func (e *Editor) handleMouse(in *engine.Input) {
	// 浮动面板模态（保存对话框/吸附设置/插件面板）：只响应面板内按钮
	if e.saveDialogOpen || e.snapPanelOpen || e.pluginPanelOpen {
		if in.MouseLeftPressed {
			switch {
			case e.saveDialogOpen:
				e.saveDialogClick(in.MouseX, in.MouseY)
			case e.snapPanelOpen:
				e.snapPanelClick(in.MouseX, in.MouseY)
			case e.pluginPanelOpen:
				e.pluginPanelClick(in.MouseX, in.MouseY)
			}
		}
		return
	}
	// 面板命中优先于视口
	if in.MouseLeftPressed {
		// 是否在视口内
		mx, my := in.MouseX-e.vpX, in.MouseY-e.vpY
		if mx >= 0 && my >= 0 && mx < e.vpW && my < e.vpH {
			e.viewportLeftDown(mx, my)
			return
		}
		// 面板
		e.panelClick(in.MouseX, in.MouseY)
		return
	}
	if in.MouseLeft {
		if e.drag != DragNone {
			mx, my := in.MouseX-e.vpX, in.MouseY-e.vpY
			e.updateDrag(mx, my)
		}
		return
	}
	if in.MouseLeftReleased {
		e.drag = DragNone
		return
	}
}

// viewportLeftDown 视口左键按下：按编辑模式分发（草图点击/骨骼拾取/gizmo/拾取）。
func (e *Editor) viewportLeftDown(mx, my int) {
	if e.editMode == EditSketch {
		e.sketchClick(mx, my)
		return
	}
	if e.editMode == EditBone {
		// 优先 gizmo（骨骼编辑），否则拾取骨骼
		if e.selBone >= 0 {
			hit := e.gizmoHit(mx, my)
			if hit != DragNone {
				e.drag = hit
				e.dragStartOK = false
				return
			}
		}
		e.bonePick(mx, my)
		return
	}
	if e.sel >= 0 {
		hit := e.gizmoHit(mx, my)
		if hit != DragNone {
			e.drag = hit
			e.dragStartOK = false
			return
		}
	}
	// 拾取
	idx := e.pickAt(mx, my)
	if idx >= 0 {
		if idx != e.sel {
			e.selPrev = e.sel // 记录上次选中（布尔 B）
		}
		e.sel = idx
		e.renaming = false
		e.fieldFocus = -1
		e.SetMessage("选中: %s", e.doc.Objs[idx].Name)
	} else {
		e.sel = -1
		e.renaming = false
		e.fieldFocus = -1
	}
}

// handleNav 视口导航：中键=orbit、右键=pan、滚轮=zoom。
func (e *Editor) handleNav(in *engine.Input) {
	mx, my := in.MouseX-e.vpX, in.MouseY-e.vpY
	inVp := mx >= 0 && my >= 0 && mx < e.vpW && my < e.vpH
	if !inVp {
		return
	}
	if in.MouseMiddle {
		if !e.orbitDrag {
			e.orbitDrag = true
			e.lastMX, e.lastMY = in.MouseX, in.MouseY
		}
		dx, dy := in.MouseX-e.lastMX, in.MouseY-e.lastMY
		e.cam.Orbit(float32(dx), float32(dy))
		e.lastMX, e.lastMY = in.MouseX, in.MouseY
		return
	}
	e.orbitDrag = false
	if in.MouseRight {
		if !e.panDrag {
			e.panDrag = true
			e.lastMX, e.lastMY = in.MouseX, in.MouseY
		}
		dx, dy := in.MouseX-e.lastMX, in.MouseY-e.lastMY
		e.cam.Pan(float32(dx), float32(dy))
		e.lastMX, e.lastMY = in.MouseX, in.MouseY
		return
	}
	e.panDrag = false
	if in.Wheel != 0 {
		f := float32(1.15)
		if in.Wheel < 0 {
			f = 1 / 1.15
		}
		e.cam.Zoom(f)
	}
}

// frameSelected 选中对象取景。
func (e *Editor) frameSelected() {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return
	}
	o := e.doc.Objs[e.sel]
	e.cam.Target = o.Pos
	e.SetMessage("取景: %s", o.Name)
}

// deleteSelected 删除选中特征。
func (e *Editor) deleteSelected() {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return
	}
	name := e.doc.Objs[e.sel].Name
	e.doc.Remove(e.sel)
	e.sel = -1
	e.drag = DragNone
	e.SetMessage("删除特征: %s", name)
}

// newDoc 新建文档（Ctrl+N）：清空场景。
func (e *Editor) newDoc() {
	e.doc = NewDocument("零件1")
	e.sel = -1
	e.drag = DragNone
	e.sketch = nil
	e.sketchHasPt0 = false
	e.sketchPt0 = Vec2{}
	e.fieldFocus = -1
	e.renaming = false
	e.SetMessage("新建文档（Ctrl+N）— 未保存的更改已丢失")
}

// duplicateSelected 复制选中特征。
func (e *Editor) duplicateSelected() {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return
	}
	cp := e.doc.Duplicate(e.sel)
	if cp != nil {
		e.sel = e.sel + 1
		e.SetMessage("复制特征: %s", cp.Name)
	}
}

// toggleVisible 切换选中特征可见性。
func (e *Editor) toggleVisible() {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return
	}
	o := e.doc.Objs[e.sel]
	o.Visible = !o.Visible
	e.SetMessage("%s %s", o.Name, map[bool]string{true: "显示", false: "隐藏"}[o.Visible])
}
