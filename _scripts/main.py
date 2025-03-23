import numpy as np
from pydub import AudioSegment
from io import BytesIO

def generate_white_noise(duration, fs=48000):
    """
    ホワイトノイズを生成する関数
    - duration: ノイズの長さ（秒）
    - fs: サンプルレート（Hz）
    
    ホワイトノイズは全周波数成分に均等なエネルギーを持つノイズです。
    ここでは -1～1 の一様分布から乱数を生成し、正規化しています。
    """
    num_samples = int(duration * fs)
    noise = np.random.uniform(-1, 1, size=num_samples)
    noise = noise / np.max(np.abs(noise))
    return noise

def generate_pink_noise(duration, fs=48000):
    """
    ピンクノイズを生成する関数
    - duration: ノイズの長さ（秒）
    - fs: サンプルレート（Hz）
    
    ピンクノイズは低周波成分が相対的に強く、高周波成分が弱い「1/f ノイズ」です。
    
    【アルゴリズムの流れ】
      1. ホワイトノイズを生成。
      2. np.fft.rfft により周波数領域へ変換。
      3. 各周波数成分に 1/(n+1) のフィルタ（n は FFT のインデックス）を乗じ、
         高周波成分を減衰させ、低周波成分を相対的に強調。
      4. np.fft.irfft で逆変換し、時系列信号に戻す。
      5. 最後に正規化して -1～1 の範囲に収める。
    """
    num_samples = int(duration * fs)
    white_noise = np.random.uniform(-1, 1, size=num_samples)
    fft_white = np.fft.rfft(white_noise)
    freqs = np.arange(len(fft_white))
    pink_filter = 1.0 / (freqs + 1)
    fft_pink = fft_white * pink_filter
    pink_noise = np.fft.irfft(fft_pink)
    pink_noise = pink_noise / np.max(np.abs(pink_noise))
    return pink_noise

def save_mp3(filename, data, fs=48000, gain_db=0):
    """
    生成したノイズ信号（-1～1 の範囲の浮動小数点配列）を MP3 ファイルとして保存する関数。
    
    手順:
      1. データを 16bit PCM (整数) に変換し、バイト列化。
      2. BytesIO 経由で AudioSegment.from_raw を用い、生データからオーディオセグメントを作成。
      3. gain_db (dB) の値を apply_gain で適用して音量を調整（正の値で増幅、負の値で減衰）。
      4. MP3 としてエクスポート。
    
    ※ pydub を利用するため、システムに ffmpeg がインストールされ、パスが通っている必要があります。
    """
    # 16bit PCM に変換
    data_int16 = np.int16(data * 32767)
    audio_bytes = data_int16.tobytes()
    
    # AudioSegment.from_raw を用いて生データからオーディオセグメントを作成
    segment = AudioSegment.from_raw(BytesIO(audio_bytes), sample_width=2, frame_rate=fs, channels=1)
    
    # gain_db が指定されている場合は音量調整
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
    
    # ピンクノイズ生成と MP3 保存
    pink_noise = generate_pink_noise(duration, fs)
    save_mp3("pink_noise_5min.mp3", pink_noise, fs)

if __name__ == '__main__':
    main()
