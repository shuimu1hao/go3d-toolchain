// main.go - 江湖行·红颜劫：回合制 RPG + Galgame 攻略五女主
package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"go2dgame/engine"
)

const (
	tilePx   = 32 // 显示 tile 大小（16x16 放大 2x）
	mapW     = 20
	mapH     = 13
	viewW    = mapW * tilePx     // 640
	viewH    = mapH * tilePx     // 416
	viewX    = (960 - viewW) / 2 // 160
	viewY    = (540 - viewH) / 2 // 62
	stepTime = 0.14              // 每格移动动画秒数
)

// Game 江湖行·红颜劫游戏主状态。
type Game struct {
	eng  *engine.Engine
	mode string // title / map / dialog / menu / battle / ending

	curMap *MapDef
	player *Player
	px, py int     // 当前 tile 坐标
	mx, my float64 // 动画插值（0~1）
	dir    string  // down up left right
	moving bool
	animT  float64

	dialog   Dialog
	menu     Menu
	battle   Battle
	giftMenu GiftMenu

	msg  string
	msgT float64

	chests      map[string]bool
	charSprites map[string]map[string]*engine.Sprite
	portraits   map[string]*engine.Sprite
	lipics      map[string]*engine.Sprite // AI 大立绘
	battleBg    *engine.Sprite
	warpSpr     [2]*engine.Sprite // 传送门两帧（闪烁）

	savePath string
	startT   time.Time
	frame    int
}

func main() {
	cfg := engine.DefaultConfig("江湖行·红颜劫 - 回合制RPG×Galgame")
	e, err := engine.New(cfg)
	if err != nil {
		fmt.Println("engine.New:", err)
		return
	}
	defer e.Close()

	if err := loadCJK("assets/fonts/NotoSansCJK-Regular.ttc", 20); err != nil {
		// 备选：直接读手机系统字体
		if err2 := loadCJK("/system/fonts/NotoSansCJK-Regular.ttc", 20); err2 != nil {
			fmt.Println("字体加载失败:", err, err2)
			return
		}
	}

	g := &Game{
		eng:         e,
		mode:        "title",
		player:      newPlayer(),
		curMap:      findMap(newPlayer().MapID),
		px:          newPlayer().PX,
		py:          newPlayer().PY,
		dir:         "down",
		chests:      map[string]bool{},
		charSprites: map[string]map[string]*engine.Sprite{},
		portraits:   map[string]*engine.Sprite{},
		lipics:      map[string]*engine.Sprite{},
		savePath:    "jianghu_rpg_save.json",
		startT:      time.Now(),
	}
	g.px = g.player.PX
	g.py = g.player.PY
	g.curMap = findMap(g.player.MapID)
	g.initAssets()
	// 传送门两帧（闪烁用）
	g.warpSpr[0] = makeWarpSprite(true)
	g.warpSpr[1] = makeWarpSprite(false)

	if err := e.Run(g); err != nil {
		fmt.Println("engine.Run:", err)
	}
}

func (g *Game) initAssets() {
	genTiles()
	initGirls()
	// 主角
	g.charSprites["hero"] = makeChar(CharColors{
		Hair: engine.Color{R: 40, G: 34, B: 38}, HairD: engine.Color{R: 26, G: 22, B: 28},
		Skin: engine.Color{R: 242, G: 202, B: 172}, Cloth: engine.Color{R: 70, G: 110, B: 190},
		ClothD: engine.Color{R: 50, G: 86, B: 152}, Pants: engine.Color{R: 60, G: 60, B: 72},
		Shoes: engine.Color{R: 80, G: 56, B: 46}, Acc: engine.Color{R: 250, G: 170, B: 200},
		Bust: 0, HairStyle: 3, // 桃夭：长发
	})
	// 五女主
	g.charSprites["qing"] = makeChar(CharColors{
		Hair: engine.Color{R: 90, G: 60, B: 44}, HairD: engine.Color{R: 66, G: 44, B: 32},
		Skin: engine.Color{R: 244, G: 204, B: 174}, Cloth: engine.Color{R: 100, G: 190, B: 110},
		ClothD: engine.Color{R: 78, G: 152, B: 88}, Pants: engine.Color{R: 200, G: 220, B: 200},
		Shoes: engine.Color{R: 140, G: 110, B: 80}, Acc: engine.Color{R: 250, G: 240, B: 200},
		Bust: 2, HairStyle: 1, // 青儿：双马尾
	})
	g.charSprites["luoyao"] = makeChar(CharColors{
		Hair: engine.Color{R: 36, G: 30, B: 34}, HairD: engine.Color{R: 24, G: 20, B: 24},
		Skin: engine.Color{R: 242, G: 202, B: 172}, Cloth: engine.Color{R: 208, G: 82, B: 70},
		ClothD: engine.Color{R: 168, G: 62, B: 54}, Pants: engine.Color{R: 88, G: 70, B: 74},
		Shoes: engine.Color{R: 70, G: 46, B: 40}, Acc: engine.Color{R: 240, G: 200, B: 90},
		Bust: 2, HairStyle: 2, // 洛瑶：高马尾
	})
	g.charSprites["yueli"] = makeChar(CharColors{
		Hair: engine.Color{R: 210, G: 210, B: 224}, HairD: engine.Color{R: 170, G: 170, B: 190},
		Skin: engine.Color{R: 238, G: 214, B: 200}, Cloth: engine.Color{R: 235, G: 240, B: 248},
		ClothD: engine.Color{R: 196, G: 204, B: 220}, Pants: engine.Color{R: 220, G: 228, B: 238},
		Shoes: engine.Color{R: 170, G: 180, B: 200}, Acc: engine.Color{R: 150, G: 200, B: 240},
		Bust: 2, HairStyle: 3, // 月璃：长发
	})
	g.charSprites["xiaoman"] = makeChar(CharColors{
		Hair: engine.Color{R: 120, G: 72, B: 40}, HairD: engine.Color{R: 90, G: 52, B: 30},
		Skin: engine.Color{R: 246, G: 208, B: 176}, Cloth: engine.Color{R: 236, G: 158, B: 66},
		ClothD: engine.Color{R: 196, G: 124, B: 50}, Pants: engine.Color{R: 150, G: 110, B: 70},
		Shoes: engine.Color{R: 96, G: 66, B: 44}, Acc: engine.Color{R: 250, G: 230, B: 150},
		Bust: 2, HairStyle: 1, // 小蛮：双马尾
	})
	g.charSprites["suxue"] = makeChar(CharColors{
		Hair: engine.Color{R: 40, G: 34, B: 50}, HairD: engine.Color{R: 26, G: 22, B: 34},
		Skin: engine.Color{R: 240, G: 206, B: 184}, Cloth: engine.Color{R: 158, G: 108, B: 190},
		ClothD: engine.Color{R: 126, G: 84, B: 156}, Pants: engine.Color{R: 96, G: 76, B: 110},
		Shoes: engine.Color{R: 80, G: 60, B: 90}, Acc: engine.Color{R: 200, G: 160, B: 240},
		Bust: 2, HairStyle: 4, // 素雪：发髻
	})
	// NPC 店小二（复用青儿配色改衣服）
	g.charSprites["npc"] = makeChar(CharColors{
		Hair: engine.Color{R: 60, G: 52, B: 44}, HairD: engine.Color{R: 42, G: 36, B: 30},
		Skin: engine.Color{R: 236, G: 196, B: 166}, Cloth: engine.Color{R: 140, G: 120, B: 100},
		ClothD: engine.Color{R: 112, G: 96, B: 80}, Pants: engine.Color{R: 92, G: 84, B: 76},
		Shoes: engine.Color{R: 60, G: 48, B: 40}, Acc: engine.Color{R: 200, G: 180, B: 140},
		Bust: 1, HairStyle: 4, // NPC 老板娘：发髻
	})
	// 女主头像
	g.portraits["qing"] = makePortrait(
		engine.Color{R: 96, G: 64, B: 48}, engine.Color{R: 70, G: 48, B: 36},
		engine.Color{R: 246, G: 206, B: 176}, engine.Color{R: 60, G: 110, B: 80},
		engine.Color{R: 244, G: 160, B: 150})
	g.portraits["luoyao"] = makePortrait(
		engine.Color{R: 40, G: 32, B: 36}, engine.Color{R: 26, G: 20, B: 24},
		engine.Color{R: 244, G: 204, B: 174}, engine.Color{R: 60, G: 80, B: 140},
		engine.Color{R: 244, G: 150, B: 140})
	g.portraits["yueli"] = makePortrait(
		engine.Color{R: 214, G: 214, B: 228}, engine.Color{R: 176, G: 176, B: 194},
		engine.Color{R: 240, G: 216, B: 202}, engine.Color{R: 90, G: 120, B: 190},
		engine.Color{R: 240, G: 180, B: 170})
	g.portraits["xiaoman"] = makePortrait(
		engine.Color{R: 124, G: 74, B: 42}, engine.Color{R: 94, G: 54, B: 32},
		engine.Color{R: 248, G: 210, B: 178}, engine.Color{R: 150, G: 100, B: 60},
		engine.Color{R: 248, G: 170, B: 150})
	g.portraits["suxue"] = makePortrait(
		engine.Color{R: 44, G: 36, B: 54}, engine.Color{R: 28, G: 22, B: 36},
		engine.Color{R: 242, G: 208, B: 186}, engine.Color{R: 120, G: 80, B: 170},
		engine.Color{R: 242, G: 170, B: 160})
	g.portraits["hero"] = makePortrait(
		engine.Color{R: 42, G: 36, B: 40}, engine.Color{R: 28, G: 24, B: 30},
		engine.Color{R: 244, G: 206, B: 176}, engine.Color{R: 250, G: 150, B: 170},
		engine.Color{R: 250, G: 170, B: 160})
	g.portraits["npc"] = makePortrait(
		engine.Color{R: 64, G: 56, B: 48}, engine.Color{R: 46, G: 40, B: 34},
		engine.Color{R: 238, G: 198, B: 168}, engine.Color{R: 90, G: 70, B: 50},
		engine.Color{R: 238, G: 170, B: 150})
	g.battleBg = makeBattleBg(640, 416)
	g.loadLipics()
}

// loadLipics 加载 AI 生成的角色大立绘（assets/portraits/*.png）。
func (g *Game) loadLipics() {
	ids := []string{"hero", "qing", "luoyao", "yueli", "xiaoman", "suxue", "zhuzhai"}
	for _, id := range ids {
		data, err := os.ReadFile("assets/portraits/" + id + ".png")
		if err != nil {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		g.lipics[id] = engine.FromImage(img)
	}
}

// Update 每帧逻辑。
func (g *Game) Update(dt float64) {
	g.frame++
	g.animT += dt
	if g.msgT > 0 {
		g.msgT -= dt
	}
	switch g.mode {
	case "title":
		g.updateTitle(dt)
	case "map":
		g.updateMap(dt)
	case "dialog":
		if g.giftMenu.active {
			g.giftUpdate(dt)
			break
		}
		g.dialog.Update(g, dt)
	case "menu":
		g.menu.Update(g, dt)
	case "battle":
		g.battle.Update(g, dt)
	case "ending":
		g.updateEnding(dt)
	}
}

// Draw 每帧渲染。
func (g *Game) Draw(c *engine.Canvas) {
	switch g.mode {
	case "title":
		g.drawTitle(c)
	case "map":
		g.drawMap(c)
		g.drawTouchOverlay(c)
	case "dialog":
		g.drawMap(c)
		if g.giftMenu.active {
			g.giftDraw(c)
		} else {
			g.dialog.Draw(c, g)
		}
	case "menu":
		g.drawMap(c)
		g.menu.Draw(c, g)
	case "battle":
		g.battle.Draw(c, g)
	case "ending":
		g.drawEnding(c)
	}
}

// ===== 标题画面 =====
// 标题按钮（触屏点按）。
var (
	titleStartBtn = Btn{340, 370, 280, 46, "开 始 游 戏"}
	titleLoadBtn  = Btn{340, 430, 280, 46, "读 取 存 档"}
)

func (g *Game) updateTitle(dt float64) {
	in := g.eng.Input()
	// 触屏兼容：点击按钮（termux-x11 触摸 = 鼠标左键）
	if in.Pressed(engine.KeyEnter) || in.Pressed(engine.KeySpace) || touchPressed(in, titleStartBtn) {
		g.startNewGame()
		return
	}
	if in.Pressed(engine.KeyChar('l')) || in.Pressed(engine.KeyChar('L')) || touchPressed(in, titleLoadBtn) {
		if g.loadSave() {
			g.msg = "读档成功！"
			g.msgT = 1.5
			g.mode = "map"
		} else {
			g.msg = "没有存档"
			g.msgT = 1.5
		}
	}
}

func (g *Game) startNewGame() {
	g.player = newPlayer()
	g.curMap = findMap("qingzhou")
	g.px, g.py = g.player.PX, g.player.PY
	g.dir = "down"
	g.mode = "map"
	g.msg = "欢迎来到江湖！按 E 与NPC对话，WASD 移动"
	g.msgT = 4
	// 初始剧情
	g.dialog.Start(g, "intro")
}

func (g *Game) drawTitle(c *engine.Canvas) {
	c.Clear(engine.Color{R: 20, G: 16, B: 30})
	// 装饰月亮
	for y := 0; y < 540; y++ {
		for x := 0; x < 960; x++ {
			dx := x - 720
			dy := y - 120
			if dx*dx+dy*dy < 3600 && dx*dx+dy*dy > 3000 {
				c.SetPixel(x, y, engine.Color{R: 220, G: 190, B: 140})
			}
		}
	}
	// 标题
	drawTextShadowCenter(c, 480, 120, "江湖行·红颜劫", engine.Color{R: 255, G: 220, B: 120})
	drawTextCenter(c, 480, 175, "回合制RPG × Galgame 攻略五女主", engine.Color{R: 200, G: 200, B: 220})
	// 五女主头像
	names := []string{"qing", "luoyao", "yueli", "xiaoman", "suxue"}
	labels := []string{"青儿", "洛瑶", "月璃", "小蛮", "素雪"}
	for i, n := range names {
		x := 260 + i*95
		y := 240
		g.portraits[n].DrawScaled(c, x, y, 2)
		drawTextCenter(c, x+32, y+70, labels[i], engine.Color{R: 220, G: 220, B: 240})
	}
	// 闪烁提示
	if int(g.animT*2)%2 == 0 {
		drawTextCenter(c, 480, 390, "按 回车/空格 开始新游戏", engine.Color{R: 255, G: 240, B: 160})
	} else {
		drawTextCenter(c, 480, 390, "按 回车/空格 开始新游戏", engine.Color{R: 160, G: 150, B: 100})
	}
	// 触屏按钮
	drawTouchBtn(c, titleStartBtn)
	drawTouchBtn(c, titleLoadBtn)
	drawTextCenter(c, 480, 500, "操作：WASD移动 E交互 回车确认 Esc菜单/返回；触屏点按钮即可", engine.Color{R: 120, G: 120, B: 140})
}

// ===== 地图模式 =====
func (g *Game) updateMap(dt float64) {
	in := g.eng.Input()
	// 动画中
	if g.moving {
		g.animT += dt
		if g.animT >= stepTime {
			g.animT = 0
			g.moving = false
			g.px, g.py = g.player.PX, g.player.PY
			// 到点检查事件（传送）
			g.checkStandEvent()
			// 遇敌
			g.checkEncounter()
		}
		return
	}
	// Esc / 触屏菜单键 打开菜单
	if in.Pressed(engine.KeyEscape) || touchPressed(in, touchMenu) {
		g.menu.Open(g)
		g.mode = "menu"
		return
	}
	// 移动
	var dx, dy int
	if in.Pressed(engine.KeyChar('w')) || in.Pressed(engine.KeyUp) {
		dy = -1
		g.dir = "up"
	}
	if in.Pressed(engine.KeyChar('s')) || in.Pressed(engine.KeyDown) {
		dy = 1
		g.dir = "down"
	}
	if in.Pressed(engine.KeyChar('a')) || in.Pressed(engine.KeyLeft) {
		dx = -1
		g.dir = "left"
	}
	if in.Pressed(engine.KeyChar('d')) || in.Pressed(engine.KeyRight) {
		dx = 1
		g.dir = "right"
	}
	// 触屏 D-pad（按住持续移动）
	if dx == 0 && dy == 0 {
		if tdx, tdy, ok := touchDir(in); ok {
			dx, dy = tdx, tdy
			switch {
			case tdy == -1:
				g.dir = "up"
			case tdy == 1:
				g.dir = "down"
			case tdx == -1:
				g.dir = "left"
			case tdx == 1:
				g.dir = "right"
			}
		}
	}
	if dx != 0 || dy != 0 {
		nx, ny := g.player.PX+dx, g.player.PY+dy
		if g.curMap.walkable(nx, ny) {
			g.player.PX, g.player.PY = nx, ny
			g.moving = true
			g.animT = 0
		} else if e := eventAt(g.curMap.ID, nx, ny); e != nil && e.Kind == "warp" {
			// 传送点在不可走处（边缘），直接传送
			g.doWarp(e)
		}
		return
	}
	// 交互 E / 空格 / 触屏交互键
	if in.Pressed(engine.KeyChar('e')) || in.Pressed(engine.KeyChar('E')) || in.Pressed(engine.KeySpace) || touchPressed(in, touchAct) {
		g.tryInteract()
	}
}

// tryInteract 面向方向的格子交互。
func (g *Game) tryInteract() {
	fx, fy := g.player.PX, g.player.PY
	switch g.dir {
	case "up":
		fy--
	case "down":
		fy++
	case "left":
		fx--
	case "right":
		fx++
	}
	// 先查事件
	if e := eventAt(g.curMap.ID, fx, fy); e != nil {
		g.handleEvent(e)
		return
	}
	// 再查脚下的 npc？某些 npc 在脚下（青儿出生点旁）
	if e := eventAt(g.curMap.ID, g.player.PX, g.player.PY); e != nil {
		g.handleEvent(e)
	}
}

func (g *Game) handleEvent(e *Event) {
	switch e.Kind {
	case "warp":
		g.doWarp(e)
	case "npc":
		if e.Param == "inn" {
			g.dialog.Start(g, "npc_inn")
		} else if gr := findGirl(e.Param); gr != nil {
			g.dialog.StartGirl(g, gr)
		} else {
			g.dialog.Start(g, "npc_"+e.Param)
		}
	case "boss":
		// 检查是否已击败
		if g.bossDefeated(e.Param) {
			g.msg = "这里的威胁已经平息了"
			g.msgT = 1.5
			return
		}
		g.battle.StartBoss(g, e.Param)
		g.mode = "battle"
	case "chest":
		if g.chests[e.MapID+"_"+fmt.Sprint(e.X)+"_"+fmt.Sprint(e.Y)] {
			g.msg = "宝箱已经被打开了"
			g.msgT = 1.5
			return
		}
		it := findItem(e.Param)
		if it != nil {
			g.player.addItem(e.Param)
			g.chests[e.MapID+"_"+fmt.Sprint(e.X)+"_"+fmt.Sprint(e.Y)] = true
			g.msg = "获得：" + it.Name + "！"
			g.msgT = 2
		}
	}
}

func (g *Game) doWarp(e *Event) {
	g.curMap = findMap(e.Param)
	g.player.MapID = e.Param
	g.player.PX, g.player.PY = e.TX, e.TY
	g.px, g.py = e.TX, e.TY
	g.moving = false
	g.msg = g.curMap.Name
	g.msgT = 1.5
}

// checkStandEvent 站在传送点/事件点上的处理。
func (g *Game) checkStandEvent() {
	if e := eventAt(g.curMap.ID, g.player.PX, g.player.PY); e != nil && e.Kind == "warp" {
		g.doWarp(e)
	}
}

func (g *Game) checkEncounter() {
	if len(g.curMap.Encounters) == 0 {
		return
	}
	if rand.Float64() < g.curMap.EncRate {
		name := g.curMap.Encounters[rand.Intn(len(g.curMap.Encounters))]
		g.battle.Start(g, name)
		g.mode = "battle"
	}
}

func (g *Game) bossDefeated(name string) bool {
	return g.player.hasItem("lingpai_" + name)
}

func (g *Game) markBossDefeated(name string) {
	g.player.addItem("lingpai_" + name)
}

// ===== 地图渲染 =====
func (g *Game) drawMap(c *engine.Canvas) {
	c.Clear(engine.Color{R: 12, G: 14, B: 20})
	// 边框
	c.FillRect(viewX-8, viewY-8, viewW+16, viewH+16, engine.Color{R: 40, G: 44, B: 58})
	c.Rect(viewX-8, viewY-8, viewW+16, viewH+16, engine.Color{R: 80, G: 86, B: 106})
	// tile
	for ty := 0; ty < mapH; ty++ {
		for tx := 0; tx < mapW; tx++ {
			t := g.curMap.tileAt(tx, ty)
			if spr, ok := tiles[t]; ok {
				spr.DrawScaled(c, viewX+tx*tilePx, viewY+ty*tilePx, 2)
			}
		}
	}
	// 事件绘制：NPC / 宝箱（门/传送门画在 tile 上）
	g.drawEventSprites(c)
	// 玩家
	g.drawPlayer(c)
	// 顶部：地图名
	drawTextShadow(c, viewX+10, viewY-30, "· "+g.curMap.Name+" ·", engine.Color{R: 240, G: 230, B: 190})
	// 右下操作提示
	drawText(c, viewX+viewW-textW("E交互 WASD移动 Esc菜单"), viewY+viewH+10, "E交互 WASD移动 Esc菜单", engine.Color{R: 160, G: 160, B: 180})
	// 消息
	if g.msgT > 0 {
		w := textW(g.msg) + 24
		x := 480 - w/2
		y := 440
		c.FillRect(x, y, w, 30, engine.Color{R: 20, G: 20, B: 30})
		c.Rect(x, y, w, 30, engine.Color{R: 120, G: 120, B: 150})
		drawText(c, x+12, y+6, g.msg, engine.Color{R: 255, G: 255, B: 240})
	}
}

// drawTouchOverlay 画地图模式的触屏虚拟键（D-pad + 交互/菜单）。
func (g *Game) drawTouchOverlay(c *engine.Canvas) {
	drawTouchBtn(c, touchUp)
	drawTouchBtn(c, touchDown)
	drawTouchBtn(c, touchLeft)
	drawTouchBtn(c, touchRight)
	drawTouchBtn(c, touchAct)
	drawTouchBtn(c, touchMenu)
	// 提示：触屏可长按方向键持续移动
	drawText(c, 8, 532, "长按方向键移动", engine.Color{R: 130, G: 130, B: 150})
}

// drawEventSprites 绘制 NPC/宝箱精灵。
func (g *Game) drawEventSprites(c *engine.Canvas) {
	// 传送门：tile 层绘制（NPC/角色之下），两帧交替闪烁
	// 用真实时间做相位（g.animT 只在移动时递增，静止会停闪）
	phase := int(time.Now().UnixNano()/250_000_000) % 2
	for i := range events {
		e := &events[i]
		if e.MapID != g.curMap.ID || e.Kind != "warp" {
			continue
		}
		w := g.warpSpr[phase]
		if w != nil {
			w.DrawScaled(c, viewX+e.X*tilePx, viewY+e.Y*tilePx, 2)
		}
	}
	// 按 y 排序保证遮挡
	type pos struct {
		x, y int
		spr  *engine.Sprite
	}
	var list []pos
	addNPC := func(ev *Event, sprName string) {
		sprs, ok := g.charSprites[sprName]
		if !ok {
			return
		}
		list = append(list, pos{ev.X, ev.Y, sprs["d0"]})
	}
	for i := range events {
		e := &events[i]
		if e.MapID != g.curMap.ID {
			continue
		}
		switch e.Kind {
		case "npc":
			addNPC(e, e.Param)
		case "boss":
			// 未击败显示威胁
			if !g.bossDefeated(e.Param) {
				// 画一个暗色剪影（用 npc 精灵变暗）
				sprs := g.charSprites["npc"]
				list = append(list, pos{e.X, e.Y, sprs["u0"]})
			}
		case "chest":
			if !g.chests[e.MapID+"_"+fmt.Sprint(e.X)+"_"+fmt.Sprint(e.Y)] {
				cb := makeChestSprite()
				list = append(list, pos{e.X, e.Y, cb})
			}
		case "warp":
			// 传送门在 tile 层单独绘制（见函数开头），不进遮挡列表
		}
	}
	// 排序画（y 小的先画）
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].y < list[i].y {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	for _, p := range list {
		p.spr.DrawScaled(c, viewX+p.x*tilePx+8, viewY+p.y*tilePx+16, 2)
	}
}

// makeChestSprite 宝箱 16x16。
func makeChestSprite() *engine.Sprite {
	s := engine.NewSprite(16, 16)
	for y := 6; y < 16; y++ {
		for x := 2; x < 14; x++ {
			s.Set(x, y, engine.Color{R: 140, G: 100, B: 56}, 255)
		}
	}
	for x := 2; x < 14; x++ {
		s.Set(x, 6, engine.Color{R: 190, G: 150, B: 90}, 255)
		s.Set(x, 7, engine.Color{R: 190, G: 150, B: 90}, 255)
	}
	s.Set(7, 8, engine.Color{R: 240, G: 210, B: 120}, 255)
	s.Set(8, 8, engine.Color{R: 240, G: 210, B: 120}, 255)
	return s
}

func (g *Game) drawPlayer(c *engine.Canvas) {
	sx, sy := g.player.PX, g.player.PY
	// 动画插值
	if g.moving {
		t := g.animT / stepTime
		if t > 1 {
			t = 1
		}
		switch g.dir {
		case "up":
			sy = g.player.PY - 1
			sx, sy = lerpPos(g.player.PX, g.player.PY, 0, -1, t)
		case "down":
			sx, sy = lerpPos(g.player.PX, g.player.PY, 0, 1, t)
		case "left":
			sx, sy = lerpPos(g.player.PX, g.player.PY, -1, 0, t)
		case "right":
			sx, sy = lerpPos(g.player.PX, g.player.PY, 1, 0, t)
		}
	}
	frame := "0"
	if g.moving {
		frame = "1"
	}
	// makeChar 生成的 key 是短名 d0/u0/l0/r0/d1...，不能直接 g.dir+frame
	dirCode := map[string]string{"up": "u", "down": "d", "left": "l", "right": "r"}[g.dir]
	spr := g.charSprites["hero"][dirCode+frame]
	if spr != nil {
		spr.DrawScaled(c, viewX+sx*tilePx+8, viewY+sy*tilePx+16, 2)
	}
}

func lerpPos(px, py, dx, dy int, t float64) (int, int) {
	return px + dx + int(float64(-dx)*(1-t)), py + dy + int(float64(-dy)*(1-t))
}

// ===== 结局 =====
func (g *Game) updateEnding(dt float64) {
	if g.eng.Input().Pressed(engine.KeyEnter) || g.eng.Input().Pressed(engine.KeySpace) || g.eng.Input().MouseLeftPressed {
		g.mode = "title"
	}
}

func (g *Game) drawEnding(c *engine.Canvas) {
	c.Clear(engine.Color{R: 24, G: 20, B: 36})
	drawTextShadowCenter(c, 480, 150, "— 全 剧 终 —", engine.Color{R: 255, G: 220, B: 120})
	if g.battle.endingText != "" {
		drawTextCenter(c, 480, 200, g.battle.endingText, engine.Color{R: 220, G: 220, B: 240})
	}
	// 列出各女主好感
	y := 260
	for _, gr := range girls {
		drawText(c, 300, y, gr.Name+"  好感度 "+fmt.Sprint(gr.Love), gr.Color)
		y += 34
	}
	drawTextCenter(c, 480, 470, "按 回车/空格 返回标题", engine.Color{R: 160, G: 160, B: 180})
}

var _ = filepath.Join
var _ = os.Getenv
