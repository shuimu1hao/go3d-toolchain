// Package engine 是 go2dgame 的 2D 游戏引擎核心。
//
// 设计目标：为 Termux 图形桌面环境（termux-x11 + XFCE）提供纯 Go、
// 零 cgo 依赖的软件渲染 2D 游戏引擎。窗口固定横屏（不做横竖屏自动切换，
// 游戏启动即横屏，其余锁定竖屏），软件渲染到像素缓冲，分块 PutImage 上屏。
package engine

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/bigreq"
	"github.com/BurntSushi/xgb/xproto"
)

// Config 描述引擎启动参数。
type Config struct {
	Title   string // 窗口标题
	Width   int    // 画布宽度（逻辑分辨率）
	Height  int    // 画布高度（逻辑分辨率）
	FPS     int    // 目标帧率
	FullWin bool   // 窗口尺寸 = 画布尺寸（默认 true）
	// 窗口位置（屏幕坐标）；<=0 时自动居中
	WinX, WinY int
}

// DefaultConfig 返回横屏默认配置：960x540@60fps，窗口位置自动计算。
func DefaultConfig(title string) Config {
	return Config{
		Title:  title,
		Width:  960,
		Height: 540,
		FPS:    60,
	}
}

// Game 是游戏主逻辑接口，由用户实现。
type Game interface {
	// Update 每帧调用一次，dt 为上一帧到本帧的秒数（已 clamp 到 0.05）。
	Update(dt float64)
	// Draw 每帧调用一次，负责把画面画到 Canvas 上。
	Draw(c *Canvas)
}

// Engine 持有 X 连接、窗口、画布与输入状态。
type Engine struct {
	Conn *xgb.Conn
	Win  xproto.Window
	GC   xproto.Gcontext

	cfg    Config
	canvas *Canvas
	input  *Input
	// 关闭标志
	closing bool
	// WM_DELETE_WINDOW atom
	wmDelete xproto.Atom
	// WM_PROTOCOLS atom（WM_DELETE 消息的 message_type）
	wmProtocols xproto.Atom
	// keycode → keysym 映射表（GetKeyboardMapping）
	keymap map[xproto.Keycode]xproto.Keysym
}

// New 创建引擎并打开 X 连接（自动探测 Termux/标准 Linux 的 socket 路径）。
func New(cfg Config) (*Engine, error) {
	conn, err := dialX()
	if err != nil {
		return nil, err
	}
	// 启用 BIG-REQUESTS（虽然分块上传不依赖它，但保留以便将来大纹理）
	if err := bigreq.Init(conn); err == nil {
		bigreq.Enable(conn).Reply()
	}
	e := &Engine{
		Conn:   conn,
		cfg:    cfg,
		keymap: map[xproto.Keycode]xproto.Keysym{},
	}
	if cfg.FPS <= 0 {
		cfg.FPS = 60
	}
	e.input = NewInput()
	e.canvas = NewCanvas(cfg.Width, cfg.Height)
	return e, nil
}

// dialX 探测 X server unix socket 并建立连接。
// Termux 无 /tmp，X socket 在 $TMPDIR/.X11-unix/ 下；标准 Linux 在 /tmp/.X11-unix/。
func dialX() (*xgb.Conn, error) {
	var tries []string
	if d := os.Getenv("DISPLAY"); d != "" && d[0] == '/' {
		tries = append(tries, d)
	}
	if t := os.Getenv("TMPDIR"); t != "" {
		tries = append(tries, t+"/.X11-unix/X0")
	}
	tries = append(tries,
		"/tmp/.X11-unix/X0",
		"/data/data/com.termux/files/usr/tmp/.X11-unix/X0",
		"/data/data/com.termux/files/usr/tmp/.X11-unix/X1",
		"/tmp/.X11-unix/X1",
		// abstract namespace（termux-x11 实测监听 @/data/data/.../.X11-unix/X0）
		"@/data/data/com.termux/files/usr/tmp/.X11-unix/X0",
		"@/tmp/.X11-unix/X0")
	var lastErr error
	for _, p := range tries {
		c, err := net.Dial("unix", p)
		if err != nil {
			lastErr = err
			continue
		}
		conn, err := xgb.NewConnNet(c)
		if err != nil {
			lastErr = err
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("cannot connect to X server (tried %v, last err: %v)", tries, lastErr)
}

// OpenWindow 创建并映射横屏窗口。窗口位置默认水平居中、垂直偏上。
// 窗口尺寸 clamp 到屏幕（手机屏幕常小于请求尺寸，避免右侧按钮被裁剪）。
func (e *Engine) OpenWindow() error {
	setup := xproto.Setup(e.Conn)
	screen := setup.DefaultScreen(e.Conn)
	w := e.cfg.Width
	h := e.cfg.Height

	// 窗口尺寸：默认 = 画布尺寸（clamp 到屏幕）
	winW, winH := w, h
	if e.cfg.FullWin {
		winW, winH = w, h
	}
	if winW > int(screen.WidthInPixels) {
		winW = int(screen.WidthInPixels)
	}
	if winH > int(screen.HeightInPixels) {
		winH = int(screen.HeightInPixels)
	}
	// clamp 后同步画布逻辑尺寸（UI 布局按实际窗口宽算，避免右缘被裁剪）
	if winW != e.canvas.W || winH != e.canvas.H {
		e.canvas.W, e.canvas.H = winW, winH
		e.canvas.WindowW, e.canvas.WindowH = winW, winH
		if len(e.canvas.Pixels) != winW*winH*4 {
			e.canvas.Pixels = make([]byte, winW*winH*4)
		}
	}

	// 自动居中：水平居中，垂直略偏上（给 XFCE 面板留出空间）
	x := e.cfg.WinX
	y := e.cfg.WinY
	if x <= 0 {
		x = (int(screen.WidthInPixels) - winW) / 2
		if x < 0 {
			x = 0
		}
	}
	if y <= 0 {
		y = (int(screen.HeightInPixels) - winH) / 3
		if y < 0 {
			y = 0
		}
	}

	wid, err := xproto.NewWindowId(e.Conn)
	if err != nil {
		return err
	}
	mask := uint32(xproto.CwBackPixel | xproto.CwEventMask)
	values := []uint32{
		0x000000, // 背景黑
		xproto.EventMaskKeyPress | xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion | xproto.EventMaskExposure |
			xproto.EventMaskStructureNotify,
	}
	xproto.CreateWindow(e.Conn, screen.RootDepth, wid, screen.Root,
		int16(x), int16(y), uint16(winW), uint16(winH), 0,
		xproto.WindowClassInputOutput, screen.RootVisual, mask, values)

	// 窗口标题
	name := e.cfg.Title
	if name == "" {
		name = "go2dgame"
	}
	xproto.ChangeProperty(e.Conn, xproto.PropModeReplace, wid,
		xproto.AtomWmName, xproto.AtomString, 8, uint32(len(name)), []byte(name))

	// WM_DELETE_WINDOW（点 X 关闭）
	delReply, _ := xproto.InternAtom(e.Conn, false, uint16(len("WM_DELETE_WINDOW")), "WM_DELETE_WINDOW").Reply()
	e.wmDelete = delReply.Atom
	protoReply, _ := xproto.InternAtom(e.Conn, false, uint16(len("WM_PROTOCOLS")), "WM_PROTOCOLS").Reply()
	e.wmProtocols = protoReply.Atom
	delData := make([]byte, 4)
	xgb.Put32(delData, uint32(e.wmDelete))
	xproto.ChangeProperty(e.Conn, xproto.PropModeReplace, wid,
		protoReply.Atom, xproto.AtomAtom, 32, 1, delData)

	// 强制置前并聚焦（游戏窗口启动即获得焦点）
	xproto.MapWindow(e.Conn, wid)

	gc, err := xproto.NewGcontextId(e.Conn)
	if err != nil {
		return err
	}
	xproto.CreateGC(e.Conn, gc, xproto.Drawable(wid), 0, nil)

	e.Win = wid
	e.GC = gc
	e.Conn.Sync()
	return nil
}

// ScreenSize 查询默认 X 屏幕尺寸（像素）。连接失败返回 ok=false。
// 用于游戏启动时把窗口尺寸 clamp 到实际屏幕，避免右侧 UI 被裁剪。
func ScreenSize() (int, int, bool) {
	conn, err := dialX()
	if err != nil {
		return 0, 0, false
	}
	defer conn.Close()
	setup := xproto.Setup(conn)
	s := setup.DefaultScreen(conn)
	return int(s.WidthInPixels), int(s.HeightInPixels), true
}

// Run 启动主循环：事件 → Update → Draw → 分块上屏，直到窗口关闭/用户退出。
// Run 会阻塞；Ctrl+C 或窗口 X 按钮可退出。
func (e *Engine) Run(game Game) error {
	if e.Win == 0 {
		if err := e.OpenWindow(); err != nil {
			return err
		}
	}

	fps := e.cfg.FPS
	if fps <= 0 {
		fps = 60
	}
	frameDur := time.Second / time.Duration(fps)

	// 首次绘制
	game.Update(0)
	game.Draw(e.canvas)
	e.canvas.Present(e)

	last := time.Now()
	for !e.closing {
		// 非阻塞收事件（每帧收满则循环，避免事件积压）
		for {
			ev, err := e.Conn.PollForEvent()
			if err != nil {
				// 连接错误：X server 挂了
				return fmt.Errorf("X event error: %v", err)
			}
			if ev == nil {
				break
			}
			e.handleEvent(ev)
		}

		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		if dt > 0.05 {
			dt = 0.05 // clamp 大帧（卡顿时防物理爆炸）
		}

		game.Update(dt)
		game.Draw(e.canvas)
		e.canvas.Present(e)
		e.input.EndFrame()

		// 帧率控制：睡到下一帧
		sleep := frameDur - time.Since(now)
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
	return nil
}

// Close 关闭窗口与连接。
func (e *Engine) Close() {
	if e.Win != 0 {
		xproto.DestroyWindow(e.Conn, e.Win)
	}
	if e.Conn != nil {
		e.Conn.Close()
	}
}

// Input 返回输入状态（每帧 Update 里读取）。
func (e *Engine) Input() *Input { return e.input }

// Canvas 返回画布（每帧 Draw 里写入）。
func (e *Engine) Canvas() *Canvas { return e.canvas }

func (e *Engine) handleEvent(ev xgb.Event) {
	switch ev := ev.(type) {
	case xproto.ExposeEvent:
		// 窗口暴露/遮挡恢复，无需特殊处理，下一帧自然重绘
	case xproto.KeyPressEvent:
		ks := e.keysym(ev.Detail)
		fmt.Printf("[EV] KeyPress keycode=%d keysym=0x%x\n", ev.Detail, ks)
		e.input.press(ks)
	case xproto.KeyReleaseEvent:
		ks := e.keysym(ev.Detail)
		e.input.release(ks)
	case xproto.ButtonPressEvent:
		fmt.Printf("[EV] ButtonPress button=%d x=%d y=%d\n", ev.Detail, ev.EventX, ev.EventY)
		e.input.mouseButton(byte(ev.Detail), true)
		e.input.mouseMove(int(ev.EventX), int(ev.EventY))
	case xproto.ButtonReleaseEvent:
		fmt.Printf("[EV] ButtonRelease button=%d\n", ev.Detail)
		e.input.mouseButton(byte(ev.Detail), false)
	case xproto.MotionNotifyEvent:
		e.input.mouseMove(int(ev.EventX), int(ev.EventY))
	case xproto.ClientMessageEvent:
		// WM_DELETE_WINDOW 协议：message_type=WM_PROTOCOLS，data[0]=WM_DELETE_WINDOW atom。
		// （旧代码只比对 ev.Type==wmDelete 永远不成立，导致点 X 关窗口后进程残留）
		if ev.Type == e.wmProtocols && len(ev.Data.Data32) > 0 && ev.Data.Data32[0] == uint32(e.wmDelete) {
			e.closing = true
		}
	case xproto.ConfigureNotifyEvent:
		// 窗口被 WM 调整尺寸时记录（不做横竖屏切换，保持逻辑分辨率不变）
		e.canvas.WindowW = int(ev.Width)
		e.canvas.WindowH = int(ev.Height)
	}
}

// keysym 返回 keycode 对应的 keysym（优先第 0 个；无映射时返回 keycode 本身）。
func (e *Engine) keysym(kc xproto.Keycode) xproto.Keysym {
	if ks, ok := e.keymap[kc]; ok {
		return ks
	}
	// 懒加载整个键盘映射表
	e.loadKeymap()
	if ks, ok := e.keymap[kc]; ok {
		return ks
	}
	return xproto.Keysym(kc)
}

func (e *Engine) loadKeymap() {
	setup := xproto.Setup(e.Conn)
	min := setup.MinKeycode
	count := int(setup.MaxKeycode-setup.MinKeycode) + 1
	reply, err := xproto.GetKeyboardMapping(e.Conn, min, byte(count)).Reply()
	if err != nil {
		return
	}
	ksymsPer := int(reply.KeysymsPerKeycode)
	base := 0
	for i := 0; i < count; i++ {
		kc := xproto.Keycode(int(min) + i)
		if base < len(reply.Keysyms) {
			e.keymap[kc] = reply.Keysyms[base]
		}
		base += ksymsPer
	}
}

// CloseRequested 返回用户是否请求退出（窗口 X 按钮）。
func (e *Engine) CloseRequested() bool { return e.closing }
