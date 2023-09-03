# Lumika

本工具基于 `zfec` `ffmpeg`, 用于将任意数据转换为以 **视频** 形式存储数据的编解码转换工具.

使用 `zfec` 为数据编码并提供冗余校验, 使用 `ffmpeg` 将编码后的数据转换为视频.

支持多线程，可一键编解码文件、自定义分割编码视频长度.

适用于文件分享，文件加密、反审查、混淆等场景.

> 按照算法存储，32x32 24fps 的视频一帧可存储 3KB 数据
> 
> 即此参数的长度为 10 小时的视频最大存储数据为 105.46875 MB

> 类似实现的项目：[Lumina](https://github.com/ERR0RPR0MPT/Lumina) [Labyrinth-go](https://github.com/ERR0RPR0MPT/Labyrinth-go), 但效率和可用性都不如 `Lumika`.
> 
> 两者在编解码上的效率对比：

```
Labyrinth: 废弃项目，生成文件容错率低，多线程，有几率无法恢复文件
Lumina: 20KB/s ~ 40KB/s，单线程，生成文件体积大，使用 QR Code 储存数据，编解码效率极低
Lumika: 500KB/s ~ 1MB/s，多线程，生成文件体积较小，可调控分段视频文件大小，编解码效率高
```

## 安装

需要安装依赖 `ffmpeg` `ffprobe` `zfec`.

### Linux

```bash
apt update
apt install ffmpeg
pip install zfec
```

### Windows

> Enter the [ffmpeg](https://ffmpeg.org/download.html) website to download the installation package and install it

## 使用

从 [Releases](https://github.com/ERR0RPR0MPT/Lumika/releases) 页面下载最新的二进制文件，放入需要编码文件的同目录下，双击运行即可.

你可以一次性选择编码本目录及其子目录下的所有`.fec`文件，也可以只选择一个文件进行编码.

同样，对于解码，程序会自动检测本目录及其子目录下的所有编码视频文件，并自动解码文件输出到同目录下.

## 效果

编码视频的大小通常在原视频大小的 5 ~ 10 倍之间(使用优化的参数)

具体取决于视频的帧率和分辨率大小，FFmpeg 的 `-preset` 等参数。

## 一些问题

在对于将编码后的视频上传到视频网站时，除了其他的限制之外，还可能会遇到一些问题，例如：

- 将编码视频上传B站后，B站会在首尾删除一些帧，在转码时长较长的视频还会在中间删帧，导致解码失败

> 在一次测试中，每个分段编码视频的长度为 05:14:01。我们使用程序自带的首尾空白帧跳过的功能规避了首尾删帧的问题，
> 但仍然观测到B站在中间删除了 2 帧的数据，而且正好是每一段编码文件的中间都丢失了 2 帧的数据，导致解码失败
> 
> 所有分段文件均在 0x026C380 处发现丢失了一帧(128byte)的数据，对于编码视频文件，在第 1-19847 帧之间完好无损，
> 之后丢失了第 19848 帧。
> 在 0x156B800 处发现丢失了第二帧(128byte)的数据，丢失的帧为第 175472 帧。
> 
> 我们向B站再次投稿了这批编码视频，我们想知道它是否是随机丢失的，对比前后两者的Hash之后，我们发现它并不是随机出现的。
> 但对于不同的文件，B站的编码器仍然可能会在不同的位置删除帧，这取决于视频的长度。
> 
> 如果必须要将编码视频上传到B站，为了避免这种错误，需要将每个切片的大小保持在 24750 KB 以下，例如B站每个BV号最大可上传100P的视频，
> 那么分100个切片的话，编码的文件大小不应该超过 230MB ，这样才能保证在上传到B站后不会出现丢帧的问题。
> 

## 高级用法

```
Usage: C:\Users\Weclont\Desktop\lumika\lumika_windows_amd64.exe [command] [options]
Double-click to run: Start via automatic mode

Commands:
add     Using ffmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.
get     Using ffmpeg to decode .mp4 FEC video files into the original files.
 Options:
 -b     The Base64 encoded JSON included message to provide decode
encode  Encode a file
 Options:
 -i     The input fec file to encode
 -s     The video size(default=32), 8-1024(must be a multiple of 8)
 -p     The output video fps setting(default=24), 1-60
 -l     The output video max segment length(seconds) setting(default=35990), 1-10^9
 -m     FFmpeg mode(default=medium): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo
decode  Decode a file
 Options:
 -i     The input file to decode
help    Show this help
```

## 许可证

[MIT License](https://github.com/ERR0RPR0MPT/Lumika/blob/main/LICENSE)
