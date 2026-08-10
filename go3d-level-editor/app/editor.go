package app

import (
	"fmt"
	"os"
	"time"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/render"
	g3e "go3deditor/app"
)

// keyF5 运行键（go2dgame 无 F5 常量）。
const keyF5 = engine.Key(0xff14)

// 布局常量（SolidWorks 风格）。
const (
	ToolbarH = 74 // 两行工具栏（28px 按钮，与建模编辑器同风格）
	TreeW    = 264
	PropW    = 304
	StatusH  = 38
)

// UI 颜色（与建模编辑器 ThemeDark 同款，统一风格）。
var (
	uiColorBg     = engine.Color{R: 18, G: 22, B: 30}
	uiColorPanel  = engine.Color{R: 46, G: 50, B: 58}
	uiColorPanel2 = engine.Color{R: 52, G: 57, B: 66}
	uiColorText   = engine.Color{R: 255, G: 255, B: 255}
	uiColorDim    = engine.Color{R: 185, G: 192, B: 200}
	uiColorBorder = engine.Color{R: 28, G: 31, B: 37}
	uiColorAccent = engine.Color{R: 90, G: 150, B: 220}
	uiColorSel    = engine.Color{R: 70, G: 110, B: 170}
	uiColorBtn    = engine.Color{R: 66, G: 72, B: 84}
	uiColorBtnHover = engine.Color{R: 84, G: 92, B: 108}
)

// Editor 关卡编辑器主状态。
type Editor struct {
	eng  *engine.Engine
	doc  *g3e.Document
	level *Level
	cam  *g3e.ViewportCam

	sel   int  // 选中实例索引
	selRes int // 资源面板选中

	// 面板状态
	resPaths   map[string]string
	resIdxs    map[string]int
	spritePaths map[string]string

	player   *Instance
	inputMap *InputMap
	playing  bool // 运行模式

	// gizmo
	drag          int
	dragAnchor    math3d.Vec3
	dragStartOK   bool
	dragAnchorAng float32
	dragAnchorDist float64
	mode          int // 0移动 1旋转 2缩放

	// 视口
	vpX, vpY, vpW, vpH int
	vpPixels []byte
	rd       *render.Renderer
	prevMX, prevMY int

	// FPS/消息
	frames int
	fps    int
	fpsCnt int
	fpsT   time.Time
	msg    string
	msgT   time.Time

	// 帮助
	showHelp bool

	// 物理
	gravity float32

	// 精灵公告板尺寸
	spriteSize float32
}

// New 创建关卡编辑器。
func New(w, h int) *Editor {
	e := &Editor{
		level:      NewLevel("关卡1"),
		sel:        -1,
		selRes:     -1,
		resPaths:   map[string]string{},
		resIdxs:    map[string]int{},
		spritePaths: map[string]string{},
		inputMap:   NewInputMap(),
		gravity:    -12,
		spriteSize: 2,
		mode:       0,
	}
	e.cam = g3e.NewViewportCam()
	return e
}

// SetEngine 绑定引擎（main 调用）。
func (e *Editor) SetEngine(eng *engine.Engine) {
	e.eng = eng
	e.fpsT = time.Now()
}

// Engine 返回引擎。
func (e *Editor) Engine() *engine.Engine { return e.eng }

// SetMessage 状态栏消息。
func (e *Editor) SetMessage(format string, args ...any) {
	e.msg = fmt.Sprintf(format, args...)
	e.msgT = time.Now()
}

// Update 每帧更新。
func (e *Editor) Update(dt float64) {
	if e.eng == nil {
		return
	}
	in := e.eng.Input()
	// 推进动画
	for _, i := range e.level.Insts {
		i.UpdateAnim(dt)
	}
	if e.playing {
		e.updatePlay(in, float32(dt))
	} else {
		e.updateEdit(in)
	}
	// FPS
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
	e.drawLeft(c)
	e.drawProps(c)
	e.drawStatus(c)
	if e.showHelp {
		e.drawHelp(c)
	}
}

// ---------- 命令 ----------

// ImportModelDoc 导入建模编辑器 JSON：所有对象成为模型资源。
func (e *Editor) ImportModelDoc(path string) error {
	doc, err := g3e.LoadDocumentFile(path)
	if err != nil {
		return err
	}
	e.doc = doc
	for idx, o := range doc.Objs {
		res := ModelResFromObject(o)
		if e.level.FindModel(res.Name) != nil {
			continue
		}
		e.level.AddModel(res)
		e.resPaths[res.Name] = path
		e.resIdxs[res.Name] = idx
	}
	e.SetMessage("导入 %d 个模型资源: %s", len(doc.Objs), path)
	return nil
}

// AddInstance 把资源实例化到场景（默认放在原点上方）。
func (e *Editor) AddInstance(resIdx int) {
	if resIdx < 0 || resIdx >= len(e.level.Models) {
		return
	}
	res := e.level.Models[resIdx]
	inst := NewInstance(res.Name+"_inst", res)
	inst.Pos = math3d.Vec3{0, 1, 0}
	inst.AnimPlaying = res.Anim != nil
	e.level.AddInstance(inst)
	e.sel = len(e.level.Insts) - 1
	e.SetMessage("添加实例: %s", inst.Name)
}

// AddSpriteInst 添加精灵实例（公告板）。
func (e *Editor) AddSpriteInst(spriteIdx int) {
	if spriteIdx < 0 || spriteIdx >= len(e.level.Sprites) {
		return
	}
	sp := e.level.Sprites[spriteIdx]
	inst := &Instance{
		Name:    sp.Name + "_精灵",
		Pos:     math3d.Vec3{0, 1, 0},
		Scale:   1,
		Visible: true,
		IsSprite: true,
		Sprite:  sp,
	}
	e.level.AddInstance(inst)
	e.sel = len(e.level.Insts) - 1
	e.SetMessage("添加精灵: %s", inst.Name)
}

// DeleteInstance 删除选中实例。
func (e *Editor) DeleteInstance() {
	if e.sel < 0 || e.sel >= len(e.level.Insts) {
		return
	}
	e.level.Insts = append(e.level.Insts[:e.sel], e.level.Insts[e.sel+1:]...)
	if e.sel >= len(e.level.Insts) {
		e.sel = len(e.level.Insts) - 1
	}
	e.SetMessage("删除实例")
}

// SetPlayer 把选中实例设为玩家（运行模式控制）。
func (e *Editor) SetPlayer() {
	if e.sel < 0 || e.sel >= len(e.level.Insts) {
		return
	}
	inst := e.level.Insts[e.sel]
	if inst.IsSprite {
		e.SetMessage("精灵不能作为玩家")
		return
	}
	for _, i := range e.level.Insts {
		i.IsPlayer = false
	}
	inst.IsPlayer = true
	inst.AnimPlaying = true
	e.player = inst
	e.SetMessage("玩家: %s", inst.Name)
}

// TogglePlay 切换运行/编辑模式。
func (e *Editor) TogglePlay() {
	e.playing = !e.playing
	if e.playing {
		for _, i := range e.level.Insts {
			i.AnimPlaying = i.Res.Anim != nil && !i.IsPlayer
		}
		e.SetMessage("运行模式（WASD 移动 空格跳跃 Esc 返回）")
	} else {
		e.SetMessage("编辑模式")
	}
}

// ImportSprite 从文件加载贴图素材。
func (e *Editor) ImportSprite(path string) error {
	sp, err := LoadSprite(path)
	if err != nil {
		return err
	}
	e.level.AddSprite(sp)
	e.spritePaths[sp.Name] = path
	e.SetMessage("加载素材: %s", sp.Name)
	return nil
}

// defaultScenePath 默认关卡文件。
func defaultScenePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "level.json"
	}
	return home + "/hermes11/go3d-level-editor/level.json"
}

// updateEdit 编辑模式输入。
func (e *Editor) updateEdit(in *engine.Input) {
	// 快捷键
	if in.Pressed(engine.KeyHelp) {
		e.showHelp = !e.showHelp
		return
	}
	if in.Pressed(keyF5) {
		e.TogglePlay()
		return
	}
	if in.Pressed(engine.KeyChar('g')) || in.Pressed(engine.KeyChar('G')) {
		e.mode = 0
		return
	}
	if in.Pressed(engine.KeyChar('r')) || in.Pressed(engine.KeyChar('R')) {
		e.mode = 1
		return
	}
	if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyChar('S')) {
		e.mode = 2
		return
	}
	if in.Pressed(engine.KeyDelete) {
		e.DeleteInstance()
		return
	}
	// Ctrl+S / Ctrl+O
	if in.Pressed(engine.KeyChar(0x13)) { // Ctrl+S
		e.saveDoc()
		return
	}
	if in.Pressed(engine.KeyChar(0x0f)) { // Ctrl+O
		e.loadDoc()
		return
	}
	// 相机导航（中键 orbit / 右键 pan / 滚轮 zoom）
	if in.MouseMiddle {
		e.cam.Orbit(float32(in.MouseX-e.prevMX)*0.01, float32(in.MouseY-e.prevMY)*0.01)
	}
	if in.MouseRight {
		e.cam.Pan(float32(in.MouseX-e.prevMX), float32(in.MouseY-e.prevMY))
	}
	if in.Wheel != 0 {
		e.cam.Zoom(float32(-in.Wheel) * 1.2)
	}
	e.prevMX, e.prevMY = in.MouseX, in.MouseY
	// 左键
	if in.MouseLeftPressed {
		// 工具栏（y < ToolbarH 优先）
		if in.MouseY < ToolbarH {
			for _, b := range uiBtns {
				if in.MouseX >= b.x && in.MouseX < b.x+b.w && in.MouseY >= b.y && in.MouseY < b.y+b.h {
					b.cb()
					return
				}
			}
		}
		// 左面板（资源/场景树）
		if in.MouseX < TreeW && in.MouseY >= ToolbarH {
			e.treeClick(in.MouseX, in.MouseY)
			return
		}
		// 视口
		mx := in.MouseX - e.vpX
		my := in.MouseY - e.vpY
		if mx >= 0 && mx < e.vpW && my >= 0 && my < e.vpH {
			// gizmo 优先
			hit := e.gizmoHit(mx, my)
			if hit >= 0 {
				e.drag = hit
				e.dragStartOK = false
				return
			}
			idx := e.pickAt(mx, my)
			if idx >= 0 {
				e.sel = idx
				e.SetMessage("选中: %s", e.level.Insts[idx].Name)
			} else {
				e.sel = -1
			}
			return
		}
		// 右侧面板按钮
		for _, b := range uiBtns {
			if in.MouseX >= b.x && in.MouseX < b.x+b.w && in.MouseY >= b.y && in.MouseY < b.y+b.h {
				b.cb()
				return
			}
		}
	}
	// 拖拽更新
	if e.drag >= 0 && (in.MouseLeft) {
		e.updateDrag(in.MouseX-e.vpX, in.MouseY-e.vpY)
	}
	if in.MouseLeftReleased {
		e.drag = -1
	}
}

// importModelDialog 载入建模编辑器 JSON（默认路径）。
func (e *Editor) importModelDialog() {
	home, err := osUserHomeDir()
	if err != nil {
		e.SetMessage("无法获取主目录")
		return
	}
	path := home + "/hermes11/go3d-toolchain/go3d-editor/scene.json"
	if err := e.ImportModelDoc(path); err != nil {
		e.SetMessage("载入失败: %v", err)
	}
}

// importSpriteDialog 载入贴图素材（默认素材目录）。
func (e *Editor) importSpriteDialog() {
	home, err := osUserHomeDir()
	if err != nil {
		e.SetMessage("无法获取主目录")
		return
	}
	path := home + "/hermes11/assets/"
	names, err := osListDir(path)
	if err != nil || len(names) == 0 {
		e.SetMessage("素材目录为空: %s", path)
		return
	}
	loaded := 0
	for _, n := range names {
		if len(n) > 4 && (n[len(n)-4:] == ".png" || n[len(n)-4:] == ".jpg" || n[len(n)-4:] == ".jpeg") {
			if err := e.ImportSprite(path + n); err == nil {
				loaded++
			}
		}
	}
	e.SetMessage("载入素材 %d 张", loaded)
}

// saveDoc 保存关卡。
func (e *Editor) saveDoc() {
	if err := e.Save(defaultScenePath()); err != nil {
		e.SetMessage("保存失败: %v", err)
		return
	}
	e.SetMessage("已保存: %s", defaultScenePath())
}

// loadDoc 载入关卡。
func (e *Editor) loadDoc() {
	if err := e.Load(defaultScenePath()); err != nil {
		e.SetMessage("载入失败: %v", err)
		return
	}
	e.sel = -1
	e.SetMessage("已载入关卡")
}

// osUserHomeDir 便捷。
func osUserHomeDir() (string, error) { return os.UserHomeDir() }

// osListDir 便捷。
func osListDir(path string) ([]string, error) {
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, en := range ents {
		if !en.IsDir() {
			names = append(names, en.Name())
		}
	}
	return names, nil
}

// renderObjects 收集渲染对象。
func (e *Editor) renderObjects() []render.Object {
	var objs []render.Object
	for _, i := range e.level.Insts {
		if !i.Visible || i.IsSprite {
			continue
		}
		objs = append(objs, i.RenderObj())
	}
	return objs
}
