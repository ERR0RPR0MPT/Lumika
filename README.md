# Lumika

![](https://raw.githubusercontent.com/ERR0RPR0MPT/Lumika/main/static/cover.jpg)

Lumika 是用于将 **任意数据** 转换为 **视频** 形式的编解码转换工具；
支持多线程, 支持帧级别的数据纠错, 可一键编解码文件、自定义分割编码视频长度；
适用于文件分享, 文件加密、反审查、混淆等场景.

## 原理

项目使用里德所罗门码为数据编码, 并对编码后视频文件的分段文件数据和帧间数据提供冗余校验的能力.

## 安装

项目除对视频处理依赖 `FFmpeg` `FFprobe` 之外, 无其他任何依赖.

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

项目默认启动 Lumika Web 作为前端页面, 使用默认的本地地址 `http://localhost:7860/ui/` 进行访问.

## 示例

### HuggingFace Space

这是项目最新版本的运行示例, 部署在 HuggingFace Space, 各项功能可正常使用.

[点此进入](https://weclont-lumika1.hf.space/ui/#/)

### Colab

[点此进入](https://colab.research.google.com/drive/1ZBJPPmn4hMF1PLD075vBPBud1G2hvm0D?usp=sharing)

### 百度飞桨

[点此进入](https://aistudio.baidu.com/projectdetail/6844423?contributionType=1&sUid=2316552&shared=1&ts=1696704155060)

## Lumika Web

这是一个单纯的前端页面, 运行在 Vercel, 仅作为示例, 并且不设置后端 API 地址将无法进行编解码操作.

> 注意：由于浏览器的限制, 该前端(https)页面无法直接访问本地(http) API (block:mixed-content). 建议使用本地 Lumika 并修改 API 进行类似的操作.

[点此进入](https://lumika.bilinside.eu.org/ui/)

## Lumika Android

现在你可以直接在 Android 上运行 Lumika 内核和 Lumika Web, 这就意味着您只需要一台基于 Android 的手机/平板
就可以对数据进行编解码.

请见 [ERR0RPR0MPT/lumika-android](https://github.com/ERR0RPR0MPT/lumika-android)

> Lumika 从 `v3.12.0` 开始完美支持 Android 运行环境.

## 哔哩源

项目目前支持哔哩哔哩的视频下载和上传, 用户可以方便地通过本项目在哔哩源上存取文件/资源.

> 项目前端页面为避免出现 CORS 跨域问题, 需要由后端转发登录请求.
> 
> 所有数据经过转发之后不会在后端留存, Cookies 会在登录后自动保存到浏览器的 localStorage 中.
> 
> 需要注意的是：在任务上传后, 虽然用户在前端无法读取 Cookies, 但后端的管理者可以在 db.json 读取明文的 Cookies.
> 
> 因此请选择可信任的后端来将任务上传到哔哩源, 或者自行搭建后端.

不久的将来可能会支持 YouTube.

> 请注意：上传到哔哩源的文件将完全公开, 项目不具备加解密的能力, 请自行套压缩包加密.
>
> 上传的任何文件资源都与开发者无关, 开发者不对用户上传的任何文件资源负责.

## Benchmark

默认配置下, 受限于视频的数据存储密度, 编码后的视频文件总大小一般为原文件的 5~10 倍.

编码速度在中端 CPU 上可达到 2M/s, 解码速度一般为编码速度的 3~5 倍(10M/s).

## 教程

[Lumika](https://www.bilibili.com/video/BV1V34y1g78f/)

[Lumika Android](https://www.bilibili.com/video/BV1MN4y1C7h8/)

推荐使用 Lumika Web 进行操作.

> 这是旧版的视频教程, 使用命令行执行, 不推荐参考.
> [BiliBili](https://www.bilibili.com/video/BV1CN4y1X7GQ/)

## 命令行

不推荐使用命令行模式.如果必须使用可以执行 `./lumika help` 或参考如下.

<details>
  <summary>点此展开</summary>

```
Usage: ./lumika.exe [command] [options]

Lumika v3.13.0
Double-click to run: Start via automatic mode

Commands:
version Output Lumika version.
web     Start Lumika Backend and Lumika Web Server, default listen on :7860.
 Options:
 -h     The host to listen on(default="")
 -p     The port to listen on(default=7860)
add     Using FFmpeg to encode zfec redundant files into .mp4 FEC video files that appear less harmful.
get     Using FFmpeg to decode .mp4 FEC video files into the original files.
encode  Encode a file
 Options:
 -i     The input fec file to encode
 -s     The video size(default=32), 8-1024(must be a multiple of 8)
 -p     The output video fps setting(default=24), 1-60
 -l     The output video max segment length(seconds) setting(default=35990), 1-10^9
 -g     The output video frame all shards(default=200), 2-256
 -k     The output video frame data shards(default=130), 2-256
 -m     FFmpeg mode(default=medium): ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo
decode  Decode a file
 Options:
 -i     The input file to decode
 -m     The output video frame all shards(default=200), 2-256
 -k     The output video frame data shards(default=130), 2-256
help    Show this help
```
</details>

## 构建

```shell
go build ./build/lumika .
```

## 开发

项目的默认配置在 `variable.go` 文件中, 可通过修改该文件来更改默认配置.

## 许可证

[MIT License](https://github.com/ERR0RPR0MPT/Lumika/blob/main/LICENSE)
