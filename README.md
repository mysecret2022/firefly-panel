# Firefly-panel (v1.0.2)

> [!IMPORTANT]
> **开源与版权声明**：
> - **原项目**：[hslr-s/sun-panel](https://github.com/hslr-s/sun-panel) (原作者：红烧猎人)
> - **二次开发作者**：roland
> - **项目声明**：本项目基于原开源项目进行二次开发与功能增强，维护者非官方维护者。完整保留原作者版权与 [MIT License](./LICENSE) 协议。
> - **主要功能**：
>   1. 【整站全量备份与无缝迁移（ZIP 打包）】：已修复容器及小窗口下显示与导出不全问题。
>   2. 【云存储与多端漫游图床】：支持 **GitHub 仓库** 与 **Cloudflare R2 (S3 通用协议)** 双引擎，支持一键全部拉取到本地或单张选择同步。
>   3. 【本地 + 云端双写机制】：默认保持关闭，用户授权后自动将新上传的图标与壁纸双写备份至云端。
>   4. 【仓库属性智能感知】：自动探测目标仓库为私有 (Private) 还是公开 (Public)，提供安全状态指引。
>   5. 【AES-256 强加密备份保障】：上传云端全站备份前**强制要求用户手动输入加密密码**，采用行业级 AES-256-GCM 算法加密，防范公开外链泄露风险。
>   6. 【修改默认初始密码】：默认账密均为 `admin` / `admin`。
>   7. 【多架构支持】：支持 `linux/amd64` (x86_64) 与 `linux/arm64` (ARM64 如各类主流 NAS) 自动化云端构建。

### 🚀 一键部署 (Docker Compose)

在您的 NAS 或服务器创建 `docker-compose.yml`：

```yaml
services:
  sun-panel:
    image: ghcr.io/mysecret2022/my-sun-panel:1.0.2
    container_name: firefly-panel
    restart: always
    ports:
      - "3002:3002"
    volumes:
      - ./conf:/app/conf
      - ./database:/app/database
      - ./uploads:/app/uploads
      - ./files:/app/files
```

执行命令一键拉起：
```bash
docker compose up -d
```
打开浏览器访问：`http://<您的IP>:3002`（初始账号密码均为：`admin` / `admin`）

---

[[ 简体中文 ]](https://doc.sun-panel.top/zh_cn/) |
[[ English ]](https://doc.sun-panel.top/)

<div align=center>

<img src="./doc/images/logo.png" width="100" height="100" />

# Sun-Panel

[![Github](https://img.shields.io/badge/Github-123456?logo=github&labelColor=242424)](https://github.com/hslr-s/sun-panel)
[![Gitee](https://img.shields.io/badge/Gitee-123456?logo=gitee&labelColor=c71d23)](https://gitee.com/hslr/sun-panel)
[![docker](https://img.shields.io/badge/docker-123456?logo=docker&logoColor=fff&labelColor=1c7aed)](https://hub.docker.com/r/hslr/sun-panel) 
[![Bilibili](https://img.shields.io/badge/Bilibili-123456?logo=bilibili&logoColor=fff&labelColor=fb7299)](https://space.bilibili.com/27407696/channel/collectiondetail?sid=2023810)
[![YouTube](https://img.shields.io/badge/YouTube-123456?logo=youtube&labelColor=ff0000)](https://www.youtube.com/channel/UCKwbFmKU25R602z6P2fgPYg)
<br>
[![GitHub User's stars](https://img.shields.io/github/stars/hslr-s%2Fsun-panel?style=flat&logo=github)](https://github.com/hslr-s/sun-panel)
[![github downloads](https://img.shields.io/github/downloads/hslr-s/sun-panel/total.svg?logo=github)](https://github.com/hslr-s/sun-panel/releases)
[![docker pulls](https://img.shields.io/docker/pulls/hslr/sun-panel.svg?logo=docker)](https://hub.docker.com/r/hslr/sun-panel)

[[ 中文文档 ]](https://sun-panel-doc.enianteam.com/zh_cn) |
[[ Document ]](https://sun-panel-doc.enianteam.com) |
[[ Demo ]](http://sunpaneldemo.enianteam.com) 

A server, NAS navigation panel, Homepage, Browser homepage.
<br>
一个服务器、NAS导航面板、Homepage、浏览器首页。

</div>


![](./doc/images/main-dark.png)

> [!IMPORTANT]
> In order to maintain the livelihood, the author added some [`PRO`] (https://pro.sun-panel.top) function, so the project temporarily entered a closed source state.; At present, the latest version of the open source is `v1.3.0`, [Please see the latest version of closed source](https://github.com/hslr-s/sun-panel/releases).; When the modular technology is developed, the separation of the PRO and the programs will be opened again, and the closed source will have no effect on ordinary users.; Let's look forward to open source again, and at the same time, we are welcome to supervise and review the security of the program.
> 
> 作者为了维持生计，增加了一些 [`PRO`](https://pro.sun-panel.top) 功能，所以项目暂时进入闭源状态。目前开源最新版本为`v1.3.0`，[闭源最新版本请查看](https://github.com/hslr-s/sun-panel/releases)。待开发出模块化技术，然后对PRO和主程序进行分离会再次开源，闭源对普通用户没有任何影响。我们一起期待再次开源吧，同时也欢迎各位大佬对程序的安全性进行监督和审查。

## 😎 Features

- 🍉 Clean interface, powerful functionality, low resource consumption
- 🍊 Easy to use, visual operation, zero-code usage
- 🍠 One-click switch between internal and external network modes
- 🍵 Supports Docker deployment (compatible with Arm systems)
- 🎪 Supports multi-account isolation
- 🎏 Supports viewing system status
- 🫙 Supports custom JS, CSS
- 🍻 Simple usage without the need to connect to an external database
- 🍾 Rich icon styles for free combination, supports [Iconify icon library](https://icon-sets.iconify.design/)
- 🚁 Supports opening small windows in the webpage (some third-party websites may block this feature)

## 🖼️ Preview Screenshots

**Various styles, freely combined**

![](./doc/images/icon-small-new.png)
![](./doc/images/transparent-info.png)
![](./doc/images/transparent-small.png)
![](./doc/images/solid-color-info.png)
![](./doc/images/full-color-small.jpg)

**Built-in small windows**

![](./doc/images/window-ssh.png)
![](./doc/images/window-xunlei.png)



## 🐳 Deployment tutorial
[Deployment Tutorial](https://sun-panel-doc.enianteam.com/usage/quick_deploy.html)

## 🍵 Donate

> Open-source development is not easy. If you feel that my project has helped you, you are welcome to [donate](./doc/donate.md) or buy me a cup of tea☕ (please leave your nickname or name in the note if possible). Your support is my motivation, thank you.


<a href="https://www.paypal.me/hslrs">
<img height="60" src="./doc/images/donate/paypal.png" target="_blank"></img> 
</a>


|   |   |
| ------------ | ------------ |
| <img height="300" src="./doc/images/donate/weixin.png"/> |  <img height="300" src="./doc/images/donate/alipay.png" /> |

## 🏖️ Communication group & community

Author：**[红烧猎人](https://blog.enianteam.com/u/sun/content/11)**

[Github Discussions](https://github.com/hslr-s/sun-panel/discussions)

QQ交流群，进不去可以点上方连接联系作者

<img src="./doc/images/qq_group_qr2.png"  height="350" />

## ❤️ Thanks

- [Roc](https://github.com/RocCheng)
- [jackloves111](https://github.com/jackloves111)
- [Rock.L](https://github.com/gitlyp)


---

[![Star History Chart](https://api.star-history.com/svg?repos=hslr-s/sun-panel&type=Date)](https://star-history.com/#hslr-s/sun-panel&Date)
