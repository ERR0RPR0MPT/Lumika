# Lumika

Lumika 是用于将 **任意数据** 转换为 **视频** 形式的编解码转换工具；
支持多线程，支持帧级别的数据纠错，可一键编解码文件、自定义分割编码视频长度；
适用于文件分享，文件加密、反审查、混淆等场景.

## 原理

项目使用里德所罗门码为数据编码，并对编码后视频文件的分段文件数据和帧间数据提供冗余校验的能力.

## 安装

项目除对视频处理依赖 `FFmpeg` `FFprobe` 之外，无其他任何依赖.

### Windows

> 进入 [FFmpeg](https://ffmpeg.org/download.html) 官方网站下载安装二进制文件
### Linux

```shell
apt update
apt install ffmpeg
```

## 使用

从 [Releases](https://github.com/ERR0RPR0MPT/Lumika/releases) 页面下载最新的二进制文件并运行.

## 前端

项目默认启动 Lumika Web 作为前端页面，使用默认的本地地址 `http://localhost:7860/ui/` 进行访问.

## 哔哩源

项目目前支持哔哩哔哩的视频下载和上传，用户可以方便地通过本项目在哔哩源上存取文件/资源.

不久的将来可能会支持 YouTube.

> 请注意：上传到哔哩源的文件将完全公开，项目不具备加解密的能力，请自行套压缩包加密.
>
> 上传的任何文件资源都与开发者无关，开发者不对用户上传的任何文件资源负责.

## Benchmark

默认配置下，受限于视频的数据存储密度，编码后的视频文件总大小一般为原文件的 5~10 倍.

编码速度在中端 CPU 上可达到 2M/s，解码速度一般为编码速度的 3~5 倍(10M/s).

## 教程

新版的前端页面还算浅显易懂，所以暂时还没有教程.

> 这是旧版的视频教程，使用命令行执行，不推荐参考.
> [BiliBili](https://www.bilibili.com/video/BV1CN4y1X7GQ/)

## 命令行

不推荐使用命令行模式.如果必须使用可以执行 `./lumika help` 或参考如下.

<details>
  <summary>点此展开</summary>

```
Usage: ./lumika.exe [command] [options]
Double-click to run: Start via automatic mode

Commands:
add     Using FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.
get     Using FFmpeg to decode .mp4 FEC video files into the original files.
 Options:
 -b     The Base64 encoded JSON included message to provide decode
encode  Encode a file
 Options:
 -i     The input fec file to encode
 -s     The video size(default=32), 8-1024(must be a multiple of 8)
 -p     The output video fps setting(default=24), 1-60
 -l     The output video max segment length(seconds) setting(default=86400), 1-10^9
 -m     FFmpeg mode(default=medium): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo
decode  Decode a file
 Options:
 -i     The input file to decode
 -g     The output video frame shards(default=10), 2-256
help    Show this help
```
</details>

## 构建

```shell
go build ./build/lumika .
```

## 开发

项目的默认配置在 `variable.go` 文件中，可通过修改该文件来更改默认配置.

## 许可证

[MIT License](https://github.com/ERR0RPR0MPT/Lumika/blob/main/LICENSE)
