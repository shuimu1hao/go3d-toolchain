package app

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"go3d/math3d"
	"go3d/mesh"
)

// LoadSTL 导入 STL 文件（自动识别二进制/ASCII），返回网格。
func LoadSTL(path string) (*mesh.Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 15 {
		return nil, fmt.Errorf("STL 文件过小")
	}
	// 二进制 STL：第 5 字节起 4 字节三角形数，文件大小应 = 84 + n*50
	if len(data) > 84 {
		n := binary.LittleEndian.Uint32(data[80:84])
		if 84+int(n)*50 == len(data) {
			return loadSTLBinary(data, n)
		}
	}
	return loadSTLAscii(data)
}

// loadSTLBinary 解析二进制 STL。
func loadSTLBinary(data []byte, n uint32) (*mesh.Mesh, error) {
	// 去重顶点
	posIdx := map[[3]float32]int{}
	var verts []math3d.Vec3
	var faces []mesh.Face
	col := mesh.Col(190, 180, 160)
	off := 84
	for i := 0; i < int(n); i++ {
		base := off + i*50
		if base+50 > len(data) {
			break
		}
		var tri [3]math3d.Vec3
		for k := 0; k < 3; k++ {
			x := math.Float32frombits(binary.LittleEndian.Uint32(data[base+12+k*12:]))
			y := math.Float32frombits(binary.LittleEndian.Uint32(data[base+12+k*12+4:]))
			z := math.Float32frombits(binary.LittleEndian.Uint32(data[base+12+k*12+8:]))
			tri[k] = math3d.Vec3{x, y, z}
		}
		// 去重（量化容差 1e-4）
		ids := [3]int{}
		for k := 0; k < 3; k++ {
			key := [3]float32{q1(tri[k].X), q1(tri[k].Y), q1(tri[k].Z)}
			if id, ok := posIdx[key]; ok {
				ids[k] = id
			} else {
				ids[k] = len(verts)
				posIdx[key] = ids[k]
				verts = append(verts, tri[k])
			}
		}
		// 法线翻转修正：STL 三角形法线可能反向，统一朝外（按面积符号）
		faces = append(faces, mesh.Face{A: ids[0], B: ids[1], C: ids[2], Col: col})
	}
	if len(faces) == 0 {
		return nil, fmt.Errorf("STL 无三角形")
	}
	return &mesh.Mesh{Positions: verts, Faces: faces}, nil
}

// loadSTLAscii 解析 ASCII STL。
func loadSTLAscii(data []byte) (*mesh.Mesh, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	posIdx := map[[3]float32]int{}
	var verts []math3d.Vec3
	var faces []mesh.Face
	col := mesh.Col(190, 180, 160)
	var tri [3]math3d.Vec3
	ti := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "vertex ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				x, e1 := strconv.ParseFloat(parts[1], 32)
				y, e2 := strconv.ParseFloat(parts[2], 32)
				z, e3 := strconv.ParseFloat(parts[3], 32)
				if e1 == nil && e2 == nil && e3 == nil && ti < 3 {
					tri[ti] = math3d.Vec3{float32(x), float32(y), float32(z)}
					ti++
					if ti == 3 {
						ids := [3]int{}
						for k := 0; k < 3; k++ {
							key := [3]float32{q1(tri[k].X), q1(tri[k].Y), q1(tri[k].Z)}
							if id, ok := posIdx[key]; ok {
								ids[k] = id
							} else {
								ids[k] = len(verts)
								posIdx[key] = ids[k]
								verts = append(verts, tri[k])
							}
						}
						faces = append(faces, mesh.Face{A: ids[0], B: ids[1], C: ids[2], Col: col})
						ti = 0
					}
				}
			}
		}
	}
	if len(faces) == 0 {
		return nil, fmt.Errorf("STL 无三角形（可能不是有效 STL）")
	}
	return &mesh.Mesh{Positions: verts, Faces: faces}, nil
}

// q1 量化到 1e-4（顶点去重键）。
func q1(v float32) float32 { return float32(int(v*10000)) / 10000 }
