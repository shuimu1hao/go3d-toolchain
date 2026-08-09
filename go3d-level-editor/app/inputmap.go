package app

import (
	"go2dgame/engine"
)

// Action 动作名称（运行模式输入）。
type Action int

const (
	ActForward Action = iota
	ActBack
	ActLeft
	ActRight
	ActJump
	ActRun
)

// ActionName 动作显示名。
func ActionName(a Action) string {
	switch a {
	case ActForward:
		return "前进"
	case ActBack:
		return "后退"
	case ActLeft:
		return "左移"
	case ActRight:
		return "右移"
	case ActJump:
		return "跳跃"
	case ActRun:
		return "加速"
	}
	return "?"
}

// InputMap 按键映射（动作 → 键）。
type InputMap struct {
	Keys map[Action]string
}

// NewInputMap 默认映射：WASD + 空格 + Shift。
func NewInputMap() *InputMap {
	return &InputMap{Keys: map[Action]string{
		ActForward: "w",
		ActBack:    "s",
		ActLeft:    "a",
		ActRight:   "d",
		ActJump:    " ",
		ActRun:     "shift",
	}}
}

// Held 动作是否按下。
func (m *InputMap) Held(in *engine.Input, a Action) bool {
	key := m.Keys[a]
	switch key {
	case "shift":
		return in.Down(engine.KeyShiftL) || in.Down(engine.KeyShiftR)
	case " ":
		return in.Down(engine.KeySpace)
	case "ctrl":
		return in.Down(engine.KeyCtrlL) || in.Down(engine.KeyCtrlR)
	case "up":
		return in.Down(engine.KeyUp)
	case "down":
		return in.Down(engine.KeyDown)
	case "left":
		return in.Down(engine.KeyLeft)
	case "right":
		return in.Down(engine.KeyRight)
	default:
		if len(key) == 1 {
			return in.Down(engine.KeyChar(key[0]))
		}
		return false
	}
}

// KeyLabel 按键显示名。
func KeyLabel(key string) string {
	names := map[string]string{
		" ": "空格", "shift": "Shift", "ctrl": "Ctrl",
		"up": "↑", "down": "↓", "left": "←", "right": "→",
	}
	if n, ok := names[key]; ok {
		return n
	}
	return key
}
