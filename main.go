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

const sampleRate = 48000 // _scripts/main.py で指定したサンプルレート

//go:embed _scripts/pink_noise_5min.mp3
var pinkNoiseData []byte

// initAudio は埋め込んだ MP3 を読み込み、ループ再生用の audio.Player を初期化して返します。
// START ボタンで Play()、STOP やタイマー終了時に Pause() を呼び出します。
func initAudio() *audio.Player {
	audioContext := audio.NewContext(sampleRate)

	// 埋め込み済みの MP3 データを bytes.Reader 経由で扱う
	reader := bytes.NewReader(pinkNoiseData)

	// MP3 を F32 版でデコード
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

	// InfiniteLoop により、音声をループ再生できるようにする
	loopStream := audio.NewInfiniteLoop(d, loopLength)

	audioPlayer, err := audioContext.NewPlayerF32(loopStream)
	if err != nil {
		log.Fatal(err)
	}
	// 初期音量は 1.0
	audioPlayer.SetVolume(1.0)
	return audioPlayer
}

func NewRoot() *Root {
	r := &Root{}
	r.countdown = 25 * time.Minute // 25分のカウントダウン
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

	startTime time.Time     // 開始時刻
	countdown time.Duration // カウントダウン時間
	remaining time.Duration // 残り時間
	running   bool          // 動作中かどうか
	paused    bool          // 一時停止中かどうか

	// バックグラウンド音声のプレイヤー（タイマー開始で再生、停止・終了で停止）
	audioPlayer *audio.Player
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
		r.remaining = 25 * time.Minute
		r.running = false
		r.paused = false
		r.startTime = time.Now()
		r.counterText.SetText(r.remainingTimeText())
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
		if r.audioPlayer != nil {
			r.audioPlayer.Play()
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
	if r.remaining < 0 {
		r.remaining = 0
		r.running = false
		// タイマー終了時に音声を停止
		if r.audioPlayer != nil {
			r.audioPlayer.Pause()
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
	r.counterText.SetText(r.remainingTimeText())
}

func (r *Root) remainingTimeText() string {
	minutes := int(r.remaining.Minutes())
	seconds := int(r.remaining.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (r *Root) Draw(context *guigui.Context, dst *ebiten.Image) {
	basicwidget.FillBackground(dst, context)
}

func main() {
	root := NewRoot()
	// バックグラウンド音声を初期化し、ルートウィジェットに設定
	root.audioPlayer = initAudio()

	op := &guigui.RunOptions{
		Title:           "ポモドーロタイマー",
		WindowMinWidth:  600,
		WindowMinHeight: 300,
	}
	if err := guigui.Run(root, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
