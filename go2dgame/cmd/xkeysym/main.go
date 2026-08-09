// xkeysym - 临时调试工具：打印 X 键盘映射，验证 keysym 表是否正常
package main

import (
	"fmt"
	"net"
	"os"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func connectX() (*xgb.Conn, error) {
	tries := []string{
		os.Getenv("TMPDIR") + "/.X11-unix/X0",
		"/data/data/com.termux/files/usr/tmp/.X11-unix/X0",
		"/data/data/com.termux/files/usr/tmp/.X11-unix/X1",
		"/tmp/.X11-unix/X1",
		"@/data/data/com.termux/files/usr/tmp/.X11-unix/X0",
		"@/tmp/.X11-unix/X0",
	}
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
	return nil, fmt.Errorf("cannot connect (last: %v)", lastErr)
}

func main() {
	conn, err := connectX()
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}
	defer conn.Close()
	setup := xproto.Setup(conn)
	min := setup.MinKeycode
	max := setup.MaxKeycode
	count := int(max-min) + 1
	reply, err := xproto.GetKeyboardMapping(conn, min, byte(count)).Reply()
	if err != nil {
		fmt.Println("GetKeyboardMapping 失败:", err)
		os.Exit(1)
	}
	ksymsPer := int(reply.KeysymsPerKeycode)
	fmt.Printf("MinKeycode=%d MaxKeycode=%d KeysymsPerKeycode=%d\n", min, max, ksymsPer)
	names := map[int]string{
		9:  "Esc", 10: "1", 11: "2", 12: "3", 13: "4", 14: "5", 15: "6",
		16: "7", 17: "8", 18: "9", 19: "0", 20: "-", 21: "=", 22: "BackSpace",
		23: "Tab", 24: "q", 25: "w", 26: "e", 27: "r", 28: "t", 29: "y",
		30: "u", 31: "i", 32: "o", 33: "p", 34: "[", 35: "]", 36: "Enter",
		37: "CtrlL", 38: "a", 39: "s", 40: "d", 41: "f", 42: "g", 43: "h",
		44: "j", 45: "k", 46: "l", 47: ";", 48: "'", 49: "`", 50: "ShiftL",
		51: "\\", 52: "z", 53: "x", 54: "c", 55: "v", 56: "b", 57: "n",
		58: "m", 59: ",", 60: ".", 61: "/", 62: "ShiftR", 63: "KP*",
		64: "AltL", 65: "Space", 66: "CapsLock", 67: "F1", 68: "F2",
		69: "F3", 70: "F4", 71: "F5", 72: "F6", 73: "F7", 74: "F8",
		75: "F9", 76: "F10", 77: "NumLock", 78: "ScrollLock",
		79: "KP7", 80: "KP8", 81: "KP9", 82: "KP-", 83: "KP4", 84: "KP5",
		85: "KP6", 86: "KP+", 87: "KP1", 88: "KP2", 89: "KP3", 90: "KP0",
		91: "KP.", 92: "KP/", 93: "F11", 94: "F12",
	}
	for i := 0; i < count; i++ {
		kc := int(min) + i
		base := i * ksymsPer
		var syms []uint32
		for j := 0; j < ksymsPer; j++ {
			if base+j < len(reply.Keysyms) {
				syms = append(syms, uint32(reply.Keysyms[base+j]))
			}
		}
		if kc == 9 || (kc >= 24 && kc <= 26) || kc == 38 || kc == 39 || kc == 40 || kc == 52 || kc == 65 || (kc >= 79 && kc <= 90) || kc == 98 || kc == 104 || kc == 111 {
			name := names[kc]
			fmt.Printf("keycode %3d (%-8s) keysyms: %v\n", kc, name, syms)
		}
	}
}
