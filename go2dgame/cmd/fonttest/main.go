package main

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	otf, err := opentype.Parse(data)
	fmt.Println("opentype.Parse err:", err)
	if err != nil {
		return
	}
	face, err := opentype.NewFace(otf, &opentype.FaceOptions{Size: 24, DPI: 72, Hinting: font.HintingFull})
	fmt.Println("NewFace err:", err)
	if err != nil {
		return
	}
	for _, r := range []rune{'侠', '江', '湖', 'A', '青', '璃'} {
		adv, err := face.GlyphAdvance(r)
		fmt.Printf("rune %q advance=%v err=%v\n", r, adv, err)
	}
	// 测量文本宽度
	adv := font.MeasureString(face, "江湖行·红颜劫")
	fmt.Println("text width:", adv)
}
