package app

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go3d/math3d"
	"go3d/mesh"
)

// SaveOBJ 把网格导出为 Wavefront OBJ（v + f，含可选颜色注释）。
func SaveOBJ(path string, m *mesh.Mesh) error {
	var sb strings.Builder
	sb.WriteString("# go3d-editor export\n")
	for _, p := range m.Positions {
		fmt.Fprintf(&sb, "v %.6f %.6f %.6f\n", p.X, p.Y, p.Z)
	}
	for i := range m.Faces {
		f := &m.Faces[i]
		fmt.Fprintf(&sb, "f %d %d %d\n", f.A+1, f.B+1, f.C+1)
		_ = f.Col
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// LoadOBJ 从 OBJ 文件加载网格（支持 v/f，忽略其他元素）。
func LoadOBJ(path string) (*mesh.Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := &mesh.Mesh{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	col := mesh.Col(170, 170, 180)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 32)
			y, _ := strconv.ParseFloat(fields[2], 32)
			z, _ := strconv.ParseFloat(fields[3], 32)
			m.Positions = append(m.Positions, math3d.Vec3{float32(x), float32(y), float32(z)})
		case "f":
			if len(fields) < 4 {
				continue
			}
			idx := func(s string) int {
				// f 格式：v / v/vt / v/vt/vn / v//vn
				part := strings.SplitN(s, "/", 2)[0]
				n, err := strconv.Atoi(part)
				if err != nil {
					return -1
				}
				if n < 0 {
					n += len(m.Positions) + 1
				}
				return n - 1
			}
			a, b, c := idx(fields[1]), idx(fields[2]), idx(fields[3])
			if a < 0 || b < 0 || c < 0 || a >= len(m.Positions) || b >= len(m.Positions) || c >= len(m.Positions) {
				continue
			}
			m.Faces = append(m.Faces, mesh.Face{A: a, B: b, C: c, Col: col})
			// 四边形拆成两个三角形
			if len(fields) >= 5 {
				d := idx(fields[4])
				if d >= 0 && d < len(m.Positions) {
					m.Faces = append(m.Faces, mesh.Face{A: a, B: c, C: d, Col: col})
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(m.Positions) == 0 {
		return nil, fmt.Errorf("OBJ 无顶点: %s", path)
	}
	return m, nil
}
