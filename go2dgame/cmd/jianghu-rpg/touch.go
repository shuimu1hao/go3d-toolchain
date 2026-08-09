// touch.go - 触屏辅助层
// termux-x11 的触摸点按 = X 鼠标左键事件（无键盘），所以各界面都要支持鼠标点击。
// 窗口逻辑分辨率 960x540；地图视图区 x∈[160,800]，两侧空白区正好放虚拟键。
package main

import "go2dgame/engine"

// Btn 一个矩形触屏按钮（960x540 逻辑坐标）。
type Btn struct {
	X, Y, W, H int
	Label      string
}

func (b Btn) Hit(x, y int) bool {
	return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
}

// 地图模式虚拟键：左侧空白区放 D-pad，右侧空白区放交互/菜单。
var (
	touchUp    = Btn{52, 440, 56, 38, "▲"}
	touchDown  = Btn{52, 486, 56, 38, "▼"}
	touchLeft  = Btn{0, 463, 52, 38, "◀"}
	touchRight = Btn{108, 463, 52, 38, "▶"}
	touchAct   = Btn{806, 462, 70, 54, "交互"}
	touchMenu  = Btn{882, 462, 70, 54, "菜单"}
)

// touchDir 按住 D-pad 时返回移动方向（按住持续移动，等价于按住 WASD）。
func touchDir(in *engine.Input) (dx, dy int, ok bool) {
	if !in.MouseLeft {
		return 0, 0, false
	}
	x, y := in.MouseX, in.MouseY
	switch {
	case touchUp.Hit(x, y):
		return 0, -1, true
	case touchDown.Hit(x, y):
		return 0, 1, true
	case touchLeft.Hit(x, y):
		return -1, 0, true
	case touchRight.Hit(x, y):
		return 1, 0, true
	}
	return 0, 0, false
}

// touchPressed 判断该按钮本帧被点按（边缘触发，用于"点一下触发一次"）。
func touchPressed(in *engine.Input, b Btn) bool {
	return in.MouseLeftPressed && b.Hit(in.MouseX, in.MouseY)
}

// drawTouchBtn 绘制触屏按钮（深色底 + 边框 + 居中标签）。
func drawTouchBtn(c *engine.Canvas, b Btn) {
	c.FillRect(b.X, b.Y, b.W, b.H, engine.Color{R: 34, G: 34, B: 52})
	c.Rect(b.X, b.Y, b.W, b.H, engine.Color{R: 170, G: 170, B: 200})
	drawTextCenter(c, b.X+b.W/2, b.Y+(b.H-lineH())/2, b.Label, engine.Color{R: 240, G: 240, B: 255})
}

// hitRow 判断点按是否命中一列等距选项行，返回行下标（-1 未命中）。
// rowY 是第 0 行顶部，rowGap 是行间距，rowH 是行高（可略大于文字高）。
func hitRow(mx, my, rowX, rowY, rowH, rowGap, n int) int {
	if n <= 0 {
		return -1
	}
	for i := 0; i < n; i++ {
		top := rowY + i*rowGap
		if my >= top && my < top+rowH {
			return i
		}
	}
	return -1
}
