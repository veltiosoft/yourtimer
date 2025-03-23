import numpy as np
from pydub import AudioSegment
from io import BytesIO

def generate_white_noise(duration, fs=48000):
    """
    ホワイトノイズを生成する関数
    - duration: ノイズの長さ（秒）
    - fs: サンプルレート（Hz）
    """
    num_samples = int(duration * fs)
    noise = np.random.uniform(-1, 1, size=num_samples)
    # 絶対最大値で正規化
    noise = noise / np.max(np.abs(noise))
    return noise

def generate_pink_noise(duration, fs=48000, norm_percentile=99.9):
    """
    ピンクノイズを生成する関数
    - duration: ノイズの長さ（秒）
    - fs: サンプルレート（Hz）
    - norm_percentile: 正規化に用いるパーセンタイル（デフォルトは 99.9）
    
    ピンクノイズは低周波成分が強い1/fノイズです。
    この実装ではホワイトノイズに対して FFT を行い、
    各周波数成分に 1/(n+1) のフィルタを適用して逆FFT しています。
    最後に 99.9パーセンタイルで正規化することで、極端なピークに左右されず、
    できるだけ信号の大部分をフルレンジに引き上げます。
    """
    num_samples = int(duration * fs)
    white_noise = np.random.uniform(-1, 1, size=num_samples)
    
    # FFT して周波数領域へ変換
    fft_white = np.fft.rfft(white_noise)
    freqs = np.arange(len(fft_white))
    # 低周波成分を強調するためのフィルタ
    pink_filter = 1.0 / (freqs + 1)
    fft_pink = fft_white * pink_filter
    
    # 逆FFT して時系列信号に戻す
    pink_noise = np.fft.irfft(fft_pink)
    
    # 99.9パーセンタイルを利用して正規化（極端なピーク値の影響を抑制）
    norm_factor = np.percentile(np.abs(pink_noise), norm_percentile)
    pink_noise = pink_noise / norm_factor
    
    # 万が一のクリッピング対策
    pink_noise = np.clip(pink_noise, -1, 1)
    return pink_noise

def save_mp3(filename, data, fs=48000, gain_db=0):
    """
    生成したノイズ信号（-1～1 の範囲の浮動小数点配列）を MP3 ファイルとして保存する関数。
    
    手順:
      1. データを 16bit PCM (整数) に変換し、バイト列化。
      2. BytesIO 経由で AudioSegment.from_raw を用い、生データからオーディオセグメントを作成。
      3. gain_db (dB) の値を apply_gain で適用して音量を調整。
      4. MP3 としてエクスポート。
    """
    # 16bit PCM に変換
    data_int16 = np.int16(data * 32767)
    audio_bytes = data_int16.tobytes()
    
    # 生データからオーディオセグメントを作成
    segment = AudioSegment.from_raw(BytesIO(audio_bytes), sample_width=2, frame_rate=fs, channels=1)
    
    # gain_db が指定されている場合は音量調整（今回はピンクノイズは 0 dB 推奨）
    if gain_db != 0:
        segment = segment.apply_gain(gain_db)
        print(f"Applied gain: {gain_db} dB")
        
    segment.export(filename, format="mp3")
    print(f"Saved {filename}")

def main():
    fs = 48000           # サンプルレート (Hz)
    duration = 5 * 60    # 5 分間 = 300 秒
    
    # ホワイトノイズ生成と MP3 保存
    white_noise = generate_white_noise(duration, fs)
    save_mp3("white_noise_5min.mp3", white_noise, fs, gain_db=-20)
    
    # ピンクノイズ生成と MP3 保存（生成時の正規化で振り幅を大きく）
    pink_noise = generate_pink_noise(duration, fs)
    save_mp3("pink_noise_5min.mp3", pink_noise, fs)

if __name__ == '__main__':
    main()
