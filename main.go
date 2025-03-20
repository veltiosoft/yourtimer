package main

import (
	"bytes"
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	_ "embed"
)

//go:embed "NotoSansJP-VariableFont_wght.ttf"
var fontData []byte

var basicFont *text.GoTextFaceSource

// レイアウトサイズの定数
const (
	// 16:9 のアスペクト比に設定
	ScreenWidth  = 960
	ScreenHeight = 540

	FontSize = 48
)

// Game は Ebiten のゲームループを管理する
type Game struct {
	startTime   time.Time
	countdown   time.Duration
	remaining   time.Duration
	isCompleted bool
}

// NewGame は新しいゲームインスタンスを作成
func NewGame() *Game {
	return &Game{
		startTime:   time.Now(),
		countdown:   10 * time.Second, // 10秒のカウントダウン
		remaining:   10 * time.Second, // 初期状態では countdown と同じ
		isCompleted: false,
	}
}

// Update はフレームごとの更新処理
func (g *Game) Update() error {
	if g.isCompleted {
		return nil
	}

	// remaining を更新
	elapsed := time.Since(g.startTime)
	g.remaining = g.countdown - elapsed

	if g.remaining <= 0 {
		g.remaining = 0
		g.isCompleted = true
	}

	return nil
}

// Draw は画面描画処理
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black) // 背景を黒に

	// mm:ss 形式でタイマーを表示
	minutes := int(g.remaining.Minutes())
	seconds := int(g.remaining.Seconds()) % 60
	msg := fmt.Sprintf("%02d:%02d", minutes, seconds)

	if g.isCompleted {
		msg = "Time's Up!"
	}

	// 画面中心より左側にタイマーを表示
	// 横方向は画面幅の1/3あたり、縦方向は画面の上部1/3あたりに配置
	textX := ScreenWidth / 4
	textY := ScreenHeight/2 - FontSize/2

	op := text.DrawOptions{}
	op.ColorScale.ScaleWithColor(color.White)
	op.GeoM.Translate(float64(textX), float64(textY)) // テキストの位置を調整
	text.Draw(screen, msg, &text.GoTextFace{
		Source: basicFont,
		Size:   FontSize,
	}, &op)
}

// Layout は画面サイズを設定
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	// フォントを初期化
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		panic(err)
	}
	basicFont = s

	// ゲームを開始
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle("Countdown Timer")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
