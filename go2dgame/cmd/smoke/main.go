// smoke: 最小 X11 窗口验证 —— 开 400x300 窗口，画渐变 + 移动方块
// 用途：验证 xgb 库在 termux-x11/XFCE 桌面环境下能否开窗、渲染、收事件
package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/bigreq"
	"github.com/BurntSushi/xgb/xproto"
)

// xConnect 探测 X server 的 unix socket 并建立连接。
// Termux 无 /tmp，X socket 在 $TMPDIR/.X11-unix/ 下；标准 Linux 在 /tmp/.X11-unix/。
func xConnect() (*xgb.Conn, error) {
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
		"/tmp/.X11-unix/X1")
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

func main() {
	if len(os.Args) > 1 && os.Args[1] == "auto" {
		// 自动模式：5 秒后自己退出（供 agent 验证用）
		go func() {
			time.Sleep(5 * time.Second)
			os.Exit(0)
		}()
	}

	conn, err := xConnect()
	if err != nil {
		fmt.Fprintln(os.Stderr, "X connect error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 启用 BIG-REQUESTS 扩展：把单请求长度上限从 64KB 提到 2GB（大帧 PutImage 必需）
	if err := bigreq.Init(conn); err != nil {
		fmt.Fprintln(os.Stderr, "bigreq.Init error:", err)
	}
	if reply, err := bigreq.Enable(conn).Reply(); err == nil {
		fmt.Println("bigreq max request length:", reply.MaximumRequestLength)
	} else {
		fmt.Fprintln(os.Stderr, "bigreq.Enable error:", err)
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)

	wid, _ := xproto.NewWindowId(conn)
	mask := uint32(xproto.CwBackPixel | xproto.CwEventMask)
	values := []uint32{
		0x202020, // 背景深灰
		xproto.EventMaskKeyPress | xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease |
			xproto.EventMaskExposure | xproto.EventMaskStructureNotify,
	}
	xproto.CreateWindow(conn, screen.RootDepth, wid, screen.Root,
		50, 50, 400, 300, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, mask, values)
	xproto.ChangeProperty(conn, xproto.PropModeReplace, wid,
		xproto.AtomWmName, xproto.AtomString, 8, uint32(len("go2dgame-smoke")), []byte("go2dgame-smoke"))
	xproto.MapWindow(conn, wid)

	// WM_DELETE_WINDOW 支持（点 X 关闭）
	wmDeleteReply, _ := xproto.InternAtom(conn, false, uint16(len("WM_DELETE_WINDOW")), "WM_DELETE_WINDOW").Reply()
	wmDelete := wmDeleteReply.Atom
	wmProtocolsReply, _ := xproto.InternAtom(conn, false, uint16(len("WM_PROTOCOLS")), "WM_PROTOCOLS").Reply()
	wmProtocols := wmProtocolsReply.Atom
	wmDeleteData := make([]byte, 4)
	xgb.Put32(wmDeleteData, uint32(wmDelete))
	xproto.ChangeProperty(conn, xproto.PropModeReplace, wid,
		wmProtocols, xproto.AtomAtom, 32, 1, wmDeleteData)

	const W, H = 400, 300
	var buf []byte

	// 创建 Graphics Context（PutImage 必需）
	gc, _ := xproto.NewGcontextId(conn)
	xproto.CreateGC(conn, gc, xproto.Drawable(wid), 0, nil)

	draw := func() {
		// 渐变背景 + 移动的彩色方块
		buf = buf[:0]
		for y := 0; y < H; y++ {
			for x := 0; x < W; x++ {
				// BGRX 格式（X 默认 visual 通常 24bit BGR）
				b := byte(x * 255 / W)
				g := byte(y * 255 / H)
				r := byte((x + y) * 128 / (W + H))
				buf = append(buf, b, g, r, 0)
			}
		}
		// 方块动画（用当前时间算位置）
		t := int(time.Now().UnixMilli()/20) % (W - 60)
		for y := 100; y < 180; y++ {
			for x := t; x < t+60; x++ {
				i := (y*W + x) * 4
				buf[i] = 0x40     // B
				buf[i+1] = 0xff   // G
				buf[i+2] = 0xff   // R  -> 黄色块
				buf[i+3] = 0
			}
		}
		xproto.PutImage(conn, xproto.ImageFormatZPixmap, xproto.Drawable(wid), gc,
			uint16(W), uint16(H), 0, 0, 0, 24, buf)
	}

	draw()
	conn.Sync()
	fmt.Println("window opened:", wid)

	// 事件循环（30fps 重绘方块）
	for {
		ev, err := conn.WaitForEvent()
		if err != nil {
			fmt.Fprintln(os.Stderr, "X event error:", err)
			return
		}
		switch ev.(type) {
		case xproto.ExposeEvent:
			draw()
		case xproto.KeyPressEvent:
			return // 任意键退出
		case xproto.ButtonPressEvent:
			return // 点击退出
		case xproto.ClientMessageEvent:
			if ce, ok := ev.(xproto.ClientMessageEvent); ok && ce.Type == wmDelete {
				return
			}
		}
		draw()
		conn.Sync()
		time.Sleep(33 * time.Millisecond)
	}
}
