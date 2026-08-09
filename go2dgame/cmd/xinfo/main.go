package main

import (
	"fmt"
	"net"
	"os"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

func main() {
	var tries []string
	if t := os.Getenv("TMPDIR"); t != "" {
		tries = append(tries, t+"/.X11-unix/X0")
	}
	tries = append(tries, "/data/data/com.termux/files/usr/tmp/.X11-unix/X0")
	var conn *xgb.Conn
	for _, p := range tries {
		c, err := net.Dial("unix", p)
		if err != nil {
			continue
		}
		conn, err = xgb.NewConnNet(c)
		if err != nil {
			continue
		}
		break
	}
	if conn == nil {
		fmt.Println("no conn")
		return
	}
	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	fmt.Printf("screen root_depth=%d root_visual=%d width=%d height=%d\n",
		screen.RootDepth, screen.RootVisual, screen.WidthInPixels, screen.HeightInPixels)
	for _, dv := range setup.PixmapFormats {
		fmt.Printf("PixmapFormat depth=%d bpp=%d\n", dv.Depth, dv.BitsPerPixel)
	}
	screenInfo := setup.Roots[0]
	fmt.Printf("allowed depths: %d\n", len(screenInfo.AllowedDepths))
	for _, d := range screenInfo.AllowedDepths {
		for _, v := range d.Visuals {
			fmt.Printf("  depth=%d visual=0x%x class=%d red=0x%x green=0x%x blue=0x%x\n",
				d.Depth, v.VisualId, v.Class, v.RedMask, v.GreenMask, v.BlueMask)
		}
	}
}
