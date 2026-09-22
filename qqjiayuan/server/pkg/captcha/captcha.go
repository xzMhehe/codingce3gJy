// Package captcha 算式图形验证码
//
// 复刻参考站 aiwapu.cn/matrix/home/captcha 的做法：服务端随机出一道加减乘算式，
// 渲染成 PNG 图片返回给前端，玩家在输入框里填「计算结果」，提交时一并校验。
//
// 项目不依赖 Redis，答案直接存在进程内存里：5 分钟过期、校验一次即作废。
package captcha

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ttl    = 5 * time.Minute // 验证码有效期
	scale  = 3               // 点阵字号放大倍数
	glyphW = 5               // 点阵字宽
	glyphH = 7               // 点阵字高
	padX   = 6               // 左右留白
	padY   = 6               // 上下留白
	gap    = 2               // 字符间距（未放大）
)

type entry struct {
	answer  string
	expires time.Time
}

var (
	mu    sync.Mutex
	store = map[string]entry{}
)

// New 生成一道算式验证码，返回验证码 ID 与 PNG 图片字节
func New() (string, []byte, error) {
	question, answer := newQuestion()
	img := render(question)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", nil, err
	}

	id := newID()
	mu.Lock()
	sweepLocked()
	store[id] = entry{answer: answer, expires: time.Now().Add(ttl)}
	mu.Unlock()
	return id, buf.Bytes(), nil
}

// Verify 校验验证码；同一个 ID 无论对错都只放行一次（防止脚本反复猜）
func Verify(id, input string) bool {
	input = strings.TrimSpace(input)
	if id == "" || input == "" {
		return false
	}
	mu.Lock()
	e, ok := store[id]
	if ok {
		delete(store, id)
	}
	mu.Unlock()
	if !ok || time.Now().After(e.expires) {
		return false
	}
	return input == e.answer
}

// newQuestion 随机出题：a+b、a-b（保证非负）、a×b
func newQuestion() (string, string) {
	a := randInt(1, 10)
	b := randInt(1, 10)
	op := []rune{'+', '-', '×'}[randInt(0, 3)]
	ans := 0
	switch op {
	case '+':
		ans = a + b
	case '-':
		if a < b {
			a, b = b, a
		}
		ans = a - b
	default:
		ans = a * b
	}
	return strconv.Itoa(a) + string(op) + strconv.Itoa(b) + "=?", strconv.Itoa(ans)
}

// render 把算式画成图片：随机深浅字色 + 干扰线 + 干扰点 + 字符上下抖动
func render(text string) *image.RGBA {
	runes := []rune(text)
	w := padX*2 + len(runes)*glyphW*scale + (len(runes)-1)*gap
	h := padY*2 + glyphH*scale
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0xF7, 0xF7, 0xF7, 0xFF}}, image.Point{}, draw.Src)

	for i := 0; i < 3; i++ {
		line(img, randInt(0, w), randInt(0, h), randInt(0, w), randInt(0, h), noiseColor())
	}
	for i := 0; i < w*h/20; i++ {
		img.SetRGBA(randInt(0, w), randInt(0, h), noiseColor())
	}

	x := padX
	for _, r := range runes {
		g, ok := font[r]
		if !ok {
			x += glyphW*scale + gap
			continue
		}
		dy := randInt(-2, 3)
		col := textColor()
		for row := 0; row < glyphH; row++ {
			for c := 0; c < glyphW; c++ {
				if g[row][c] == '#' {
					fillRect(img, x+c*scale, padY+row*scale+dy, scale, scale, col)
				}
			}
		}
		x += glyphW*scale + gap
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, col color.RGBA) {
	b := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx < b.Min.X || xx >= b.Max.X || yy < b.Min.Y || yy >= b.Max.Y {
				continue
			}
			img.SetRGBA(xx, yy, col)
		}
	}
}

// line 画一条干扰线（Bresenham）
func line(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		b := img.Bounds()
		if x0 >= b.Min.X && x0 < b.Max.X && y0 >= b.Min.Y && y0 < b.Max.Y {
			img.SetRGBA(x0, y0, col)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

var textPalette = []color.RGBA{
	{0x20, 0x20, 0x20, 0xFF},
	{0x8B, 0x1A, 0x1A, 0xFF},
	{0x0B, 0x4F, 0x8A, 0xFF},
	{0x1B, 0x5E, 0x20, 0xFF},
	{0x6A, 0x1B, 0x9A, 0xFF},
}

var noisePalette = []color.RGBA{
	{0xB0, 0xB0, 0xB0, 0xFF},
	{0xC8, 0xD8, 0xE8, 0xFF},
	{0xE0, 0xC8, 0xC8, 0xFF},
}

func textColor() color.RGBA  { return textPalette[randInt(0, len(textPalette))] }
func noiseColor() color.RGBA { return noisePalette[randInt(0, len(noisePalette))] }

// randInt 返回 [min, max) 的随机整数
func randInt(min, max int) int {
	if max <= min {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	if err != nil {
		return min
	}
	return min + int(n.Int64())
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// sweepLocked 顺手清理过期验证码（调用方需已持锁）
func sweepLocked() {
	if len(store) == 0 {
		return
	}
	now := time.Now()
	for k, v := range store {
		if now.After(v.expires) {
			delete(store, k)
		}
	}
}

// font 5×7 点阵字模（'#' 为实心点）
var font = map[rune][]string{
	'0': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	'3': {"#####", "...#.", "..#..", "...#.", "....#", "#...#", ".###."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {"..##.", ".#...", "#....", "####.", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", ".####", "....#", "...#.", ".##.."},
	'+': {".....", "..#..", "..#..", "#####", "..#..", "..#..", "....."},
	'-': {".....", ".....", ".....", "#####", ".....", ".....", "....."},
	'×': {".....", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "....."},
	'=': {".....", ".....", "#####", ".....", "#####", ".....", "....."},
	'?': {".###.", "#...#", "....#", "...#.", "..#..", ".....", "..#.."},
}
