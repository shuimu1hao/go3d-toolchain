// Pong：go2dgame 引擎示例游戏 —— 双人乒乓。
//
// 操作：左拍 W/S 上下，右拍 ↑/↓（键盘）；空格发球/重开；Esc 退出。
// 首次得分方发球；先到 7 分获胜。
package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"go2dgame/engine"
)

const (
	winScore = 7
	ballR    = 10
	padW     = 14
	padH     = 90
	padGap   = 40 // 球拍距左右边缘距离
	speedUp  = 1.06
	maxSpeed = 900.0
)

// Ball 弹球状态（位置/速度）。
type Ball struct {
	X, Y   float64
	VX, VY float64
}

// Game 乒乓游戏主状态。
type Game struct {
	eng      *engine.Engine
	leftY    float64 // 左拍中心 Y
	rightY   float64 // 右拍中心 Y
	ball     Ball
	scoreL   int
	scoreR   int
	serving  bool // 等待发球
	serveDir float64
	over     bool
	winner   string
}

func main() {
	cfg := engine.DefaultConfig("PONG - go2dgame demo")
	e, err := engine.New(cfg)
	if err != nil {
		fmt.Println("engine.New:", err)
		return
	}
	defer e.Close()

	g := &Game{eng: e}
	g.resetBall(1)

	if err := e.Run(g); err != nil {
		fmt.Println("engine.Run:", err)
	}
}

func (g *Game) resetBall(dir float64) {
	g.ball = Ball{
		X:  float64(g.eng.Canvas().W) / 2,
		Y:  float64(g.eng.Canvas().H) / 2,
		VX: dir * 320,
		VY: float64(rand.Intn(200)-100) / 100 * 220,
	}
	g.serving = true
	g.serveDir = dir
}

func (g *Game) Update(dt float64) {
	in := g.eng.Input()
	c := g.eng.Canvas()

	if g.over {
		if in.Pressed(engine.KeySpace) || in.Pressed(engine.KeyEnter) {
			g.scoreL, g.scoreR = 0, 0
			g.over = false
			g.resetBall(1)
		}
		return
	}

	// 球拍移动
	const padSpeed = 520.0
	if in.Down(engine.KeyChar('w')) || in.Down(engine.KeyUp) {
		g.leftY -= padSpeed * dt
	}
	if in.Down(engine.KeyChar('s')) || in.Down(engine.KeyDown) {
		g.leftY += padSpeed * dt
	}
	if in.Down(engine.KeyChar('k')) || in.Down(engine.KeyUpDup) {
		g.rightY -= padSpeed * dt
	}
	if in.Down(engine.KeyChar('j')) || in.Down(engine.KeyDownDup) {
		g.rightY += padSpeed * dt
	}
	// 夹住边界
	maxY := float64(c.H - padH/2)
	if g.leftY < padH/2 {
		g.leftY = padH / 2
	}
	if g.leftY > maxY {
		g.leftY = maxY
	}
	if g.rightY < padH/2 {
		g.rightY = padH / 2
	}
	if g.rightY > maxY {
		g.rightY = maxY
	}

	// 发球
	if g.serving {
		if in.Pressed(engine.KeySpace) || in.Pressed(engine.KeyEnter) {
			g.serving = false
			g.ball.VX = g.serveDir * 320
			g.ball.VY = (rand.Float64() - 0.5) * 200
		}
		return
	}

	// 球移动
	g.ball.X += g.ball.VX * dt
	g.ball.Y += g.ball.VY * dt

	// 上下墙反弹
	if g.ball.Y-ballR < 0 {
		g.ball.Y = ballR
		g.ball.VY = math.Abs(g.ball.VY)
	}
	if g.ball.Y+ballR > float64(c.H) {
		g.ball.Y = float64(c.H) - ballR
		g.ball.VY = -math.Abs(g.ball.VY)
	}

	// 球拍碰撞
	// 左拍
	padLX := float64(padGap + padW/2)
	if g.ball.X-ballR < padLX+float64(padW/2) && g.ball.X+ballR > padLX-float64(padW/2) &&
		g.ball.VX < 0 && g.ball.Y > g.leftY-float64(padH)/2-ballR && g.ball.Y < g.leftY+float64(padH)/2+ballR {
		g.ball.X = padLX + float64(padW/2) + ballR
		g.ball.VX = math.Abs(g.ball.VX) * speedUp
		g.ball.VY += (g.ball.Y - g.leftY) / (float64(padH)/2) * 180
		g.clampSpeed()
	}
	// 右拍
	padRX := float64(c.W - padGap - padW/2)
	if g.ball.X+ballR > padRX-float64(padW/2) && g.ball.X-ballR < padRX+float64(padW/2) &&
		g.ball.VX > 0 && g.ball.Y > g.rightY-float64(padH)/2-ballR && g.ball.Y < g.rightY+float64(padH)/2+ballR {
		g.ball.X = padRX - float64(padW/2) - ballR
		g.ball.VX = -math.Abs(g.ball.VX) * speedUp
		g.ball.VY += (g.ball.Y - g.rightY) / (float64(padH)/2) * 180
		g.clampSpeed()
	}

	// 出界得分
	if g.ball.X < -50 {
		g.scoreR++
		g.afterScore(-1)
	}
	if g.ball.X > float64(c.W)+50 {
		g.scoreL++
		g.afterScore(1)
	}
}

func (g *Game) clampSpeed() {
	sp := math.Sqrt(g.ball.VX*g.ball.VX + g.ball.VY*g.ball.VY)
	if sp > maxSpeed {
		k := maxSpeed / sp
		g.ball.VX *= k
		g.ball.VY *= k
	}
}

func (g *Game) afterScore(dir float64) {
	if g.scoreL >= winScore {
		g.over = true
		g.winner = "LEFT PLAYER WINS"
		return
	}
	if g.scoreR >= winScore {
		g.over = true
		g.winner = "RIGHT PLAYER WINS"
		return
	}
	g.resetBall(dir)
}

func (g *Game) Draw(c *engine.Canvas) {
	c.Clear(engine.Color{R: 18, G: 22, B: 30})

	// 中线
	for y := 8; y < c.H-8; y += 28 {
		c.FillRect(c.W/2-3, y, 6, 16, engine.Color{R: 60, G: 70, B: 90})
	}

	// 球拍
	padLX := padGap
	padRX := c.W - padGap - padW
	c.FillRect(padLX, int(g.leftY)-padH/2, padW, padH, engine.ColCyan)
	c.FillRect(padRX, int(g.rightY)-padH/2, padW, padH, engine.ColPink)

	// 球（带尾光）
	if !g.serving && !g.over {
		tx := g.ball.X - g.ball.VX*0.02
		ty := g.ball.Y - g.ball.VY*0.02
		c.Circle(int(tx), int(ty), ballR-3, engine.Color{R: 60, G: 90, B: 120})
	}
	c.Circle(int(g.ball.X), int(g.ball.Y), ballR, engine.ColWhite)

	// 计分
	c.TextCentered(c.W/2, 26, fmt.Sprintf("%d   %d", g.scoreL, g.scoreR), engine.ColWhite, 3)

	// 发球提示
	if g.serving && !g.over {
		c.TextCentered(c.W/2, c.H/2+60, "SPACE to serve", engine.ColYellow, 2)
	}

	// 胜利画面
	if g.over {
		c.FillRect(c.W/2-240, c.H/2-50, 480, 100, engine.Color{R: 30, G: 36, B: 50})
		c.Rect(c.W/2-240, c.H/2-50, 480, 100, engine.ColYellow)
		c.TextCentered(c.W/2, c.H/2-18, g.winner, engine.ColYellow, 2)
		c.TextCentered(c.W/2, c.H/2+22, "SPACE / ENTER to restart", engine.ColGray, 1)
	}

	// 底部操作提示
	c.TextCentered(c.W/2, c.H-18, "W/S or Arrows: move | Space: serve | Esc: quit",
		engine.ColGray, 1)
}

var _ = time.Now
