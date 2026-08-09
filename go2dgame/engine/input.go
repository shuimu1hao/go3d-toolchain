package engine

import "github.com/BurntSushi/xgb/xproto"

// 常用 keysym 常量（X11 keysym 值的子集）。
// keysym 是语义键码，与物理键盘布局无关；WASD/方向键/回车等都能稳定识别。
const (
	KeyLeft    = xproto.Keysym(0xff51)
	KeyUp      = xproto.Keysym(0xff52)
	KeyRight   = xproto.Keysym(0xff53)
	KeyDown    = xproto.Keysym(0xff54)
	KeyEnter   = xproto.Keysym(0xff0d)
	KeyEscape  = xproto.Keysym(0xff1b)
	KeySpace   = xproto.Keysym(0x0020)
	KeyTab     = xproto.Keysym(0xff09)
	KeyBack    = xproto.Keysym(0xff08)
	KeyShiftL  = xproto.Keysym(0xffe1)
	KeyShiftR  = xproto.Keysym(0xffe2)
	KeyCtrlL   = xproto.Keysym(0xffe3)
	KeyCtrlR   = xproto.Keysym(0xffe4)
	KeyReturn  = xproto.Keysym(0xff8d) // 数字小键盘回车
	KeyDelete  = xproto.Keysym(0xffff) // Delete
	KeyHelp    = xproto.Keysym(0xffbe) // F1
	KeyUpDup   = xproto.Keysym(0xff55) // keypad Up
	KeyLeftDup = xproto.Keysym(0xff51) // keypad Left
	KeyDownDup = xproto.Keysym(0xff56) // keypad Down
	KeyRightDup = xproto.Keysym(0xff53) // keypad Right
)

// Key 是按键的 keysym 别名（方便用户代码里直接比较）。
type Key = xproto.Keysym

// Input 保存键盘与鼠标的当前状态。
// 事件循环每帧把 X 事件写入这里；游戏代码在 Update/Draw 里轮询。
type Input struct {
	// 当前按住的键集合（keysym → 结构体）
	held map[Key]bool
	// 本帧新按下的键集合（只在按下那一帧为 true）
	pressed map[Key]bool
	// 本帧新松开的键集合
	released map[Key]bool

	// 鼠标
	MouseX, MouseY int
	// 鼠标按键：1=左 2=中 3=右
	MouseLeft, MouseMiddle, MouseRight bool
	// 鼠标本帧刚按下/释放（边缘触发）
	MouseLeftPressed, MouseRightPressed   bool
	MouseLeftReleased, MouseRightReleased bool
	// 滚轮增量：本帧上滚 +1 / 下滚 -1（每次滚动累加，EndFrame 清零）
	Wheel int
}

// NewInput 创建输入状态。
func NewInput() *Input {
	return &Input{
		held:    make(map[Key]bool),
		pressed: make(map[Key]bool),
		released: make(map[Key]bool),
	}
}

// press 记录按键按下（事件循环调用）。
func (in *Input) press(k Key) {
	// 无条件标记 pressed：若此前某次 KeyRelease 丢失导致 held 卡 true，
	// 带守卫的写法会让该键永远失效（2026-08-09 实测 xdotool 快速按键丢 release）。
	in.pressed[k] = true
	in.held[k] = true
}

// release 记录按键松开（事件循环调用）。
func (in *Input) release(k Key) {
	in.held[k] = false
	in.released[k] = true
}

// mouseButton 记录鼠标按键（事件循环调用）。
// X11 滚轮：Button4=上滚，Button5=下滚（作为 Wheel 增量记录，不改变按键状态）。
func (in *Input) mouseButton(btn byte, down bool) {
	if btn == 4 {
		if down {
			in.Wheel += 1
		}
		return
	}
	if btn == 5 {
		if down {
			in.Wheel -= 1
		}
		return
	}
	switch btn {
	case 1:
		if down {
			in.MouseLeft = true
			in.MouseLeftPressed = true
		} else {
			in.MouseLeft = false
			in.MouseLeftReleased = true
		}
	case 2:
		in.MouseMiddle = down
	case 3:
		if down {
			in.MouseRight = true
			in.MouseRightPressed = true
		} else {
			in.MouseRight = false
			in.MouseRightReleased = true
		}
	}
}

// mouseMove 记录鼠标位置（事件循环调用）。
func (in *Input) mouseMove(x, y int) {
	in.MouseX = x
	in.MouseY = y
}

// Down 返回键是否按住。
func (in *Input) Down(k Key) bool { return in.held[k] }

// Pressed 返回键是否本帧刚按下（边缘触发，适合“按键一次触发一次”）。
func (in *Input) Pressed(k Key) bool { return in.pressed[k] }

// Released 返回键是否本帧刚松开。
func (in *Input) Released(k Key) bool { return in.released[k] }

// AnyPressed 返回本帧是否有任意键按下（用于“按任意键继续”）。
func (in *Input) AnyPressed() bool { return len(in.pressed) > 0 }

// 便捷：字符键（a-z, A-Z, 0-9 等直接比较 keysym 数值）
// keysym 的小写字母就是 0x61-0x7a，数字就是 0x30-0x39。
func KeyChar(ch byte) Key { return Key(ch) }

// EndFrame 在每帧结束时清理边缘触发状态（引擎主循环自动调用）。
func (in *Input) EndFrame() {
	in.pressed = make(map[Key]bool)
	in.released = make(map[Key]bool)
	in.MouseLeftPressed = false
	in.MouseRightPressed = false
	in.MouseLeftReleased = false
	in.MouseRightReleased = false
	in.Wheel = 0
}
