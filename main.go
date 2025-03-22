// このプログラムは guigui/example/counter/main.go を改変して作成したものです。
// 元のソースコードとの著作権は以下の通りです。
// https://github.com/hajimehoshi/guigui/blob/3a01a55446f47a457eb3f07164247e922cb1df63/example/counter/main.go
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Hajime Hoshi

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
)

func NewRoot() *Root {
	r := &Root{}

	r.countdown = 25 * time.Minute // 25分のカウントダウン
	r.remaining = 25 * time.Minute
	r.running = false
	return r
}

type Root struct {
	guigui.RootWidget

	resetButton basicwidget.TextButton
	// タイマー停止ボタン
	stopButton basicwidget.TextButton
	// タイマー開始ボタン
	startButton basicwidget.TextButton
	counterText basicwidget.Text

	startTime time.Time     // 開始時刻
	countdown time.Duration // カウントダウン時間
	remaining time.Duration // 残り時間
	running   bool          // 動作中かどうか
	paused    bool          // 一時停止中かどうか
}

func (r *Root) Layout(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
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

	r.resetButton.SetText("Reset")
	r.resetButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.resetButton.SetOnUp(func() {
		fmt.Println("Reset")
		// カウントダウンをリセット
		r.remaining = 25 * time.Minute
		r.running = false
		r.paused = false
		r.startTime = time.Now() // 開始時刻もリセット
		r.counterText.SetText(r.remainingTimeText())
	})
	{
		p := guigui.Position(r)
		_, h := r.Size(context)
		p.X += basicwidget.UnitSize(context)
		p.Y += h - 2*basicwidget.UnitSize(context)
		guigui.SetPosition(&r.resetButton, p)
		appender.AppendChildWidget(&r.resetButton)
	}

	r.stopButton.SetText("STOP")
	r.stopButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.stopButton.SetOnUp(func() {
		// カウントダウンを停止
		r.running = false
		r.paused = true
	})
	{
		p := guigui.Position(r)
		w, h := r.Size(context)
		p.X += w - 7*basicwidget.UnitSize(context)
		p.Y += h - 2*basicwidget.UnitSize(context)
		guigui.SetPosition(&r.stopButton, p)
		appender.AppendChildWidget(&r.stopButton)
	}

	r.startButton.SetText("START")
	r.startButton.SetWidth(6 * basicwidget.UnitSize(context))
	r.startButton.SetOnUp(func() {
		// カウントダウンを開始
		r.running = true

		// 一時停止から再開の場合は、調整された開始時刻を設定する
		if r.paused {
			r.startTime = time.Now().Add(-r.countdown + r.remaining)
			r.paused = false
		} else {
			r.startTime = time.Now()
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

	// remaining を更新
	elapsed := time.Since(r.startTime)
	r.remaining = r.countdown - elapsed
	if r.remaining < 0 {
		r.remaining = 0
		r.running = false
	}
	r.setCounterText()
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
	op := &guigui.RunOptions{
		Title:           "ポモドーロタイマー", // タイトルをアプリの目的に合わせて変更
		WindowMinWidth:  600,
		WindowMinHeight: 300,
	}
	if err := guigui.Run(NewRoot(), op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
