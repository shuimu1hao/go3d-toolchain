// go3d-demo-game：纯代码 3D 小游戏（不依赖编辑器）
// WASD 移动收集金币，撞到箱子掉血，F 冲刺，Esc 退出。
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/mesh"
	"go3d/render"
)

const (
	W, H    = 720, 480
	SPEED   = 3.5
	DASH    = 8.0
	MAXHP   = 5
	COIN_N  = 6
	BOX_N   = 5
	ARENA   = 9.0
	PICKUP  = 0.9
	HURT    = 0.8
)

var (
	cGround = mesh.Col(60, 90, 60)
	cPlayer = mesh.Col(80, 200, 220)
	cCoin   = mesh.Col(240, 210, 60)
	cBox    = mesh.Col(190, 70, 60)
)

// Game 3D 收集游戏主状态。
type Game struct {
	eng     *engine.Engine
	player  math3d.Vec3
	hp      int
	score   int
	coins   []math3d.Vec3
	boxes   []math3d.Vec3
	time    float64
	over    bool
	frames  int
}

func main() {
	cfg := engine.DefaultConfig("go3d-demo-game")
	cfg.Width, cfg.Height = W, H
	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}
	defer eng.Close()

	g := &Game{eng: eng, player: math3d.Vec3{0, 0.5, 0}, hp: MAXHP}
	r := rand.New(rand.NewSource(7))
	for i := 0; i < COIN_N; i++ {
		g.coins = append(g.coins, math3d.Vec3{r.Float32()*ARENA*2 - ARENA, 0.4, r.Float32()*ARENA*2 - ARENA})
	}
	for i := 0; i < BOX_N; i++ {
		g.boxes = append(g.boxes, math3d.Vec3{r.Float32()*ARENA*2 - ARENA, 0.5, r.Float32()*ARENA*2 - ARENA})
	}

	eng.Run(g)
}

// Update 实现 engine.Game。
func (g *Game) Update(dt float64) {
	g.frames++
	in := g.eng.Input()
	if in.Pressed(engine.KeyEscape) {
		os.Exit(0)
	}
	if g.over {
		if in.Pressed(engine.KeyChar('r')) || in.Pressed(engine.KeyChar('R')) {
			g.player = math3d.Vec3{0, 0.5, 0}
			g.hp = MAXHP
			g.score = 0
			g.over = false
		}
		return
	}
	// 移动
	spd := float32(SPEED)
	if in.Down(engine.KeyChar('f')) || in.Down(engine.KeyChar('F')) {
		spd = DASH
	}
	dx, dz := float32(0), float32(0)
	dtf := float32(dt)
	if in.Down(engine.KeyChar('w')) || in.Down(engine.KeyChar('W')) {
		dz -= spd * dtf
	}
	if in.Down(engine.KeyChar('s')) || in.Down(engine.KeyChar('S')) {
		dz += spd * dtf
	}
	if in.Down(engine.KeyChar('a')) || in.Down(engine.KeyChar('A')) {
		dx -= spd * dtf
	}
	if in.Down(engine.KeyChar('d')) || in.Down(engine.KeyChar('D')) {
		dx += spd * dtf
	}
	g.player.X += dx
	g.player.Z += dz
	// 边界
	if g.player.X < -ARENA {
		g.player.X = -ARENA
	}
	if g.player.X > ARENA {
		g.player.X = ARENA
	}
	if g.player.Z < -ARENA {
		g.player.Z = -ARENA
	}
	if g.player.Z > ARENA {
		g.player.Z = ARENA
	}
	// 收集金币
	left := g.coins[:0]
	for _, c := range g.coins {
		dx := g.player.X - c.X
		dz := g.player.Z - c.Z
		if dx*dx+dz*dz < PICKUP*PICKUP {
			g.score += 10
			continue
		}
		left = append(left, c)
	}
	g.coins = left
	// 撞箱子
	for _, b := range g.boxes {
		dx := g.player.X - b.X
		dz := g.player.Z - b.Z
		dist := float32(math.Sqrt(float64(dx*dx + dz*dz)))
		if dist < HURT && dist > 0.001 {
			g.hp--
			if g.hp <= 0 {
				g.over = true
			}
			// 完全弹开（分离到碰撞半径外，避免嵌进箱子）
			g.player.X = b.X + dx/dist*(HURT+0.1)
			g.player.Z = b.Z + dz/dist*(HURT+0.1)
			// 边界
			if g.player.X < -ARENA {
				g.player.X = -ARENA
			}
			if g.player.X > ARENA {
				g.player.X = ARENA
			}
			if g.player.Z < -ARENA {
				g.player.Z = -ARENA
			}
			if g.player.Z > ARENA {
				g.player.Z = ARENA
			}
		}
	}
	// 金币清空胜利
	if len(g.coins) == 0 {
		g.score += 50
		g.over = true
	}
}

// Draw 实现 engine.Game。
func (g *Game) Draw(c *engine.Canvas) {
	// 相机（俯视跟随）
	cam := &render.Camera{
		Pos:        g.player.Add(math3d.Vec3{0, 14, 7}),
		LookTarget: &g.player,
		FOV:        0.9,
		Near:       0.1,
		Far:        40,
	}
	c.Clear(engine.Color{R: 24, G: 28, B: 36})
	rd := render.NewRenderer(W, H)
	pixels := make([]byte, W*H*4)
	rd.Clear(pixels)
	objs := []render.Object{
		// 地面（薄板 18x0.2x18）
		{Mesh: thinBox(18, 0.2, 18), Pos: math3d.Vec3{0, -0.2, 0}, ColorTint: &cGround},
	}
	// 玩家（青）
	objs = append(objs, render.Object{Mesh: mesh.Cube(1.0), Pos: g.player, ColorTint: &cPlayer})
	// 金币（黄，旋转动画）
	spin := float32(g.frames) * 0.05
	for _, c := range g.coins {
		objs = append(objs, render.Object{Mesh: mesh.Cylinder(0.3, 0.15, 12), Pos: c, RotY: spin, ColorTint: &cCoin})
	}
	// 箱子（红）
	for _, b := range g.boxes {
		objs = append(objs, render.Object{Mesh: mesh.Cube(1.0), Pos: b, ColorTint: &cBox})
	}
	rd.Render(pixels, cam, objs)
	// 调试：统计非背景像素 + 中心像素
	np := 0
	for i := 0; i < len(pixels); i += 16 {
		if pixels[i+2] != 18 || pixels[i+1] != 22 || pixels[i] != 30 {
			np++
		}
	}
	ci := (H/2*W + W/2) * 4
	cp := fmt.Sprintf("中心(%d,%d)=%d,%d,%d", W/2, H/2, pixels[ci+2], pixels[ci+1], pixels[ci])
	if g.frames%30 == 0 {
		fmt.Printf("[dbg] frames=%d player=%v objs=%d np=%d %s\n", g.frames, g.player, len(objs), np, cp)
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := (y*W + x) * 4
			c.SetPixel(x, y, engine.Color{R: pixels[i+2], G: pixels[i+1], B: pixels[i]})
		}
	}
	// HUD
	hud := fmt.Sprintf("HP %d   分数 %d   金币 %d", g.hp, g.score, len(g.coins))
	drawText(c, 12, 12, hud, engine.Color{R: 255, G: 255, B: 255})
	if g.over {
		if len(g.coins) == 0 {
			drawText(c, W/2-60, H/2, "WIN! R RESTART", engine.Color{R: 255, G: 220, B: 80})
		} else {
			drawText(c, W/2-60, H/2, "GAME OVER! R RESTART", engine.Color{R: 255, G: 100, B: 80})
		}
	} else {
		drawText(c, 12, H-24, "WASD MOVE  F DASH  COLLECT COINS", engine.Color{R: 180, G: 190, B: 210})
	}
}

// 5x7 位图字体（数字 + 大写字母 + 常用符号，HUD 用）。
var font5x7 = map[byte][7]byte{
	'0': {0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	'1': {0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'2': {0x0E, 0x11, 0x01, 0x02, 0x04, 0x08, 0x1F},
	'3': {0x1F, 0x02, 0x04, 0x02, 0x01, 0x11, 0x0E},
	'4': {0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	'5': {0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	'6': {0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	'7': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	'8': {0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	'9': {0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	'A': {0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'B': {0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	'C': {0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	'D': {0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	'E': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	'F': {0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	'G': {0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0F},
	'H': {0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	'I': {0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	'J': {0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	'K': {0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	'L': {0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	'M': {0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	'N': {0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	'O': {0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'P': {0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	'Q': {0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	'R': {0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	'S': {0x0F, 0x10, 0x10, 0x0E, 0x01, 0x01, 0x1E},
	'T': {0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	'U': {0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	'V': {0x11, 0x11, 0x11, 0x11, 0x11, 0x0A, 0x04},
	'W': {0x11, 0x11, 0x11, 0x15, 0x15, 0x15, 0x0A},
	'X': {0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	'Y': {0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	'Z': {0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	' ': {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	'-': {0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00},
	'.': {0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x0C},
	'!': {0x04, 0x04, 0x04, 0x04, 0x04, 0x00, 0x04},
}

func drawText(c *engine.Canvas, x, y int, s string, col engine.Color) {
	cx := x
	for i := 0; i < len(s); i++ {
		drawChar(c, cx, y, s[i], col)
		cx += 7
	}
}

func drawChar(c *engine.Canvas, x, y int, ch byte, col engine.Color) {
	pat, ok := font5x7[ch]
	if !ok {
		pat = font5x7[' ']
	}
	for row := 0; row < 7; row++ {
		b := pat[row]
		for cx2 := 0; cx2 < 5; cx2++ {
			if b&(1<<uint(4-cx2)) != 0 {
				c.SetPixel(x+cx2, y+row, col)
			}
		}
	}
}

func meshColor(r, g, b uint8) mesh.Color { return mesh.Col(r, g, b) }

// thinBox 生成薄盒网格（尺寸任意，非等比）。
func thinBox(w, h, d float32) *mesh.Mesh {
	x, y, z := w/2, h/2, d/2
	verts := []math3d.Vec3{
		{-x, -y, -z}, {x, -y, -z}, {x, y, -z}, {-x, y, -z},
		{-x, -y, z}, {x, -y, z}, {x, y, z}, {-x, y, z},
	}
	faces := [][3]int{
		{0, 1, 2}, {0, 2, 3}, {4, 5, 6}, {4, 6, 7},
		{0, 1, 5}, {0, 5, 4}, {1, 2, 6}, {1, 6, 5},
		{2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7},
	}
	col := mesh.Col(200, 200, 200)
	m := &mesh.Mesh{Positions: verts}
	for _, f := range faces {
		m.Faces = append(m.Faces, mesh.Face{A: f[0], B: f[1], C: f[2], Col: col})
	}
	return m
}
