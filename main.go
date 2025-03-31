// このプログラムは guigui/example/counter/main.go を改変して作成したものです。
// 元のソースコードとの著作権は以下の通りです。
// https://github.com/hajimehoshi/guigui/blob/3a01a55446f47a457eb3f07164247e922cb1df63/example/counter/main.go
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Hajime Hoshi

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
)

// TODO(zztkm): sampleRate を mp3 ファイルから取得するようにする
const sampleRate = 48000 // _scripts/main.py で指定したサンプルレート

const (
	timeupPlayerDefaultVolume = 0.2 // 効果音の音量
)

// TODO: 背景音楽を指定してアプリを起動することができるようにする
//go:embed Morning.mp3
var pinkNoiseData []byte

// 効果音ファイル timeup.mp3 を埋め込み
//
//go:embed timeup.mp3
var timeupData []byte

// initAudio は背景音用の MP3 を読み込み、ループ再生用の audio.Player を初期化します。
func initAudio(ctx *audio.Context) *audio.Player {
	reader := bytes.NewReader(pinkNoiseData)
	// TODO(zztkm): d.SampleRate() でサンプルレートを取得できる、この場合 audio.Context の
	// 生成前に mp3.DecodeF32 を実行する必要がある
	d, err := mp3.DecodeF32(reader)
	if err != nil {
		log.Fatal(err)
	}
	// ループ端部のノイズを抑えるため、全体の長さより約 0.1[s]分少なく設定する
	const extraTimeSeconds = 0.1
	extraBytes := int64(float64(sampleRate*4) * extraTimeSeconds)
	loopLength := d.Length() - extraBytes
	if loopLength < 0 {
		loopLength = d.Length()
	}
	loopStream := audio.NewInfiniteLoop(d, loopLength)
	audioPlayer, err := ctx.NewPlayerF32(loopStream)
	if err != nil {
		log.Fatal(err)
	}
	audioPlayer.SetVolume(1.0)
	return audioPlayer
}

// initTimeupAudio は効果音用の MP3 を読み込み、1 回再生用の audio.Player を初期化します。
func initTimeupAudio(ctx *audio.Context) *audio.Player {
	reader := bytes.NewReader(timeupData)
	d, err := mp3.DecodeF32(reader)
	if err != nil {
		log.Fatal(err)
	}
	audioPlayer, err := ctx.NewPlayerF32(d)
	if err != nil {
		log.Fatal(err)
	}
	audioPlayer.SetVolume(1.0)
	return audioPlayer
}

func NewRoot() *Root {
	r := &Root{}
	// 初期状態は作業セッション：25分
	r.workSession = true
	r.countdown = 25 * time.Minute
	r.remaining = 25 * time.Minute
	r.running = false
	r.volume = 1.0 // 初期音量 1.0
	return r
}

type Root struct {
	guigui.RootWidget

	resetButton basicwidget.TextButton // リセットボタン
	stopButton  basicwidget.TextButton // タイマー停止ボタン
	startButton basicwidget.TextButton // タイマー開始ボタン

	counterText basicwidget.Text

	// 音量調整用ウィジェット
	volUpButton   basicwidget.TextButton
	volDownButton basicwidget.TextButton
	volumeText    basicwidget.Text
	volume        float64

	startTime time.Time     // セッション開始時刻
	countdown time.Duration // 現在のセッションのカウントダウン時間
	remaining time.Duration // 残り時間
	running   bool          // 動作中かどうか
	paused    bool          // 一時停止中かどうか

	// セッション種別: true は作業（25分）、false は休憩（5分）
	workSession bool

	// 背景音と効果音のプレイヤー
	audioPlayer  *audio.Player // 背景音（ピンクノイズ）用
	timeupPlayer *audio.Player // タイマー終了時の効果音用
}

func (r *Root) Layout(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	// カウンタ表示のレイアウト
	{
		w, h := r.Size(context)
		w -= 2 * basicwidget.UnitSize(context)
		h -= 4 * basicwidget.UnitSize(context)
		r.counterText.SetSize(w, h)
		p := guigui.Position(r)
		p.X += basicwidget.UnitSize(context)
		p.Y += basicwidget.UnitSize(context)
		guigui.SetPosition(&r.counterText, p)
		appender.AppendChildWidget(&r.counterText)
	}

	// Reset ボタン
	r.resetButton.SetText("Reset")
	r.resetButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.resetButton.SetOnUp(func() {
		fmt.Println("Reset")
		r.workSession = true
		r.countdown = 25 * time.Minute
		r.remaining = 25 * time.Minute
		r.running = false
		r.paused = false
		r.startTime = time.Now()
		r.counterText.SetText(r.formatRemainingTime())
		if r.audioPlayer != nil {
			r.audioPlayer.Pause()
			r.audioPlayer.Rewind()
		}
	})
	{
		p := guigui.Position(r)
		_, h := r.Size(context)
		p.X += basicwidget.UnitSize(context)
		p.Y += h - 2*basicwidget.UnitSize(context)
		guigui.SetPosition(&r.resetButton, p)
		appender.AppendChildWidget(&r.resetButton)
	}

	// STOP ボタン
	r.stopButton.SetText("STOP")
	r.stopButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.stopButton.SetOnUp(func() {
		r.running = false
		r.paused = true
		if r.audioPlayer != nil {
			r.audioPlayer.Pause()
		}
	})
	{
		p := guigui.Position(r)
		w, h := r.Size(context)
		p.X += w - 7*basicwidget.UnitSize(context)
		p.Y += h - 2*basicwidget.UnitSize(context)
		guigui.SetPosition(&r.stopButton, p)
		appender.AppendChildWidget(&r.stopButton)
	}

	// START ボタン
	r.startButton.SetText("START")
	r.startButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.startButton.SetOnUp(func() {
		r.running = true
		if r.paused {
			r.startTime = time.Now().Add(-r.countdown + r.remaining)
			r.paused = false
		} else {
			r.startTime = time.Now()
		}
		// 作業セッション時はBGMを再生、休憩時はBGM不要
		if r.workSession {
			if r.audioPlayer != nil {
				r.audioPlayer.Play()
			}
		} else {
			if r.audioPlayer != nil {
				r.audioPlayer.Pause()
			}
		}
	})
	{
		p := guigui.Position(r)
		w, h := r.Size(context)
		p.X += w - int(13.5*float64(basicwidget.UnitSize(context)))
		p.Y += h - 2*basicwidget.UnitSize(context)
		guigui.SetPosition(&r.startButton, p)
		appender.AppendChildWidget(&r.startButton)
	}

	// 音量調整ウィジェットのレイアウト
	// Vol- ボタン
	r.volDownButton.SetText("Vol-")
	r.volDownButton.SetWidth(4 * basicwidget.UnitSize(context))
	r.volDownButton.SetOnUp(func() {
		r.volume -= 0.1
		if r.volume < 0.0 {
			r.volume = 0.0
		}
		if r.audioPlayer != nil {
			r.audioPlayer.SetVolume(r.volume)
		}
		r.volumeText.SetText(fmt.Sprintf("Vol: %.1f", r.volume))
	})
	{
		w, _ := r.Size(context)
		p := guigui.Position(r)
		p.X = w - 15*basicwidget.UnitSize(context)
		p.Y = basicwidget.UnitSize(context)
		guigui.SetPosition(&r.volDownButton, p)
		appender.AppendChildWidget(&r.volDownButton)
	}

	// Vol+ ボタン
	r.volUpButton.SetText("Vol+")
	r.volUpButton.SetWidth(4 * basicwidget.UnitSize(context))
	r.volUpButton.SetOnUp(func() {
		r.volume += 0.1
		if r.volume > 1.0 {
			r.volume = 1.0
		}
		if r.audioPlayer != nil {
			r.audioPlayer.SetVolume(r.volume)
		}
		r.volumeText.SetText(fmt.Sprintf("Vol: %.1f", r.volume))
	})
	{
		w, _ := r.Size(context)
		p := guigui.Position(r)
		p.X = w - 9*basicwidget.UnitSize(context)
		p.Y = basicwidget.UnitSize(context)
		guigui.SetPosition(&r.volUpButton, p)
		appender.AppendChildWidget(&r.volUpButton)
	}

	// 音量表示テキスト
	r.volumeText.SetText(fmt.Sprintf("Vol: %.1f", r.volume))
	r.volumeText.SetSelectable(true)
	r.volumeText.SetBold(true)
	r.volumeText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	r.volumeText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.volumeText.SetScale(1.5)
	{
		w, _ := r.Size(context)
		p := guigui.Position(r)
		p.X = w - 20*basicwidget.UnitSize(context)
		p.Y = basicwidget.UnitSize(context)
		guigui.SetPosition(&r.volumeText, p)
		appender.AppendChildWidget(&r.volumeText)
	}
}

func (r *Root) Update(context *guigui.Context) error {
	if !r.running {
		r.setCounterText()
		guigui.Enable(&r.startButton)
		guigui.Disable(&r.stopButton)
		return nil
	}
	guigui.Enable(&r.stopButton)
	guigui.Disable(&r.startButton)

	// 残り時間の更新
	elapsed := time.Since(r.startTime)
	r.remaining = r.countdown - elapsed

	// セッション終了時は自動で次のセッションに切り替える
	if r.remaining <= 0 {
		// 効果音を再生
		if r.timeupPlayer != nil {
			r.timeupPlayer.Rewind()
			r.timeupPlayer.Play()
		}

		if r.workSession {
			// 作業セッション終了 → 休憩セッション開始（BGMは不要）
			r.workSession = false
			r.countdown = 5 * time.Minute
			r.remaining = r.countdown
			r.startTime = time.Now()
			if r.audioPlayer != nil {
				r.audioPlayer.Pause() // BGM停止
			}
		} else {
			// 休憩セッション終了 → 作業セッション開始
			r.workSession = true
			r.countdown = 25 * time.Minute
			r.remaining = r.countdown
			r.startTime = time.Now()
			if r.audioPlayer != nil {
				r.audioPlayer.Play() // 作業セッションはBGM再生
			}
		}
	}
	r.setCounterText()

	if r.volume == 1.0 {
		guigui.Disable(&r.volUpButton)
		guigui.Enable(&r.volDownButton)
	} else if r.volume == 0.0 {
		guigui.Disable(&r.volDownButton)
		guigui.Enable(&r.volUpButton)
	} else {
		guigui.Enable(&r.volUpButton)
		guigui.Enable(&r.volDownButton)
	}
	return nil
}

func (r *Root) setCounterText() {
	r.counterText.SetSelectable(true)
	r.counterText.SetBold(true)
	r.counterText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	r.counterText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	r.counterText.SetScale(4)
	r.counterText.SetText(r.formatRemainingTime())
}

func (r *Root) formatRemainingTime() string {
	minutes := int(r.remaining.Minutes())
	seconds := int(r.remaining.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (r *Root) Draw(context *guigui.Context, dst *ebiten.Image) {
	basicwidget.FillBackground(dst, context)
}

func main() {
	ctx := audio.NewContext(sampleRate)
	root := NewRoot()
	root.audioPlayer = initAudio(ctx)
	root.timeupPlayer = initTimeupAudio(ctx)
	root.timeupPlayer.SetVolume(timeupPlayerDefaultVolume)

	op := &guigui.RunOptions{
		Title:           "ポモドーロタイマー",
		WindowMinWidth:  600,
		WindowMinHeight: 300,
	}
	if err := guigui.Run(root, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
