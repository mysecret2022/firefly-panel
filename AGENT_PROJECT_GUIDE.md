# Firefly-panel v1.0.2 智能体项目维护与架构指南

> **开源声明与致敬**：  
> 本项目基于 [hslr-s/sun-panel](https://github.com/hslr-s/sun-panel) (MIT License) 进行深度二次开发与扩展，维护者：**Roland**。保留原作者版权与署名。

---

## 1. 版本概览 (v1.0.2)

`v1.0.2` 完成了视觉品牌定制、系统弹窗交互升级、严格权限隔离、图库管理全套分类与批量操作、云端漫游图床分类与限流删除、管理员一键审批中心以及图标合规修复。

### 核心新增与优化特性清单：
1. **视觉品牌与图标合规化**：
   - 默认桌面壁纸替换为流萤主题背景（`defaultBackground.webp`）。
   - 网站图标通过 Python Pillow 转换为标准 Windows 多分辨率 `.ico` 二进制文件（内嵌 `16x16`, `32x32`, `48x48`, `64x64`）以及 `favicon.png`，彻底根除浏览器图标格式兼容性问题。
   - 关于页面 Logo 替换为“我真厉害”专属徽标（`about_logo.png`），规范二次开发作者为 **Roland**（R 大写），GitHub 仓库链接设置为新窗口外链跳转。
   - 登录页面初始账密提示维持原版 `admin` / `admin`。
   - 系统应用菜单顺序调整：【关于】模块在侧边栏中精确移动到【云存储与漫游】正下方。

2. **系统应用弹窗交互增强 (Draggable & Resizable)**：
   - `RoundCardModal` 组件底层升级：
     - **自由拖拽移动**：鼠标按住卡片标题栏可自由平滑拖拽移动弹窗位置，双击标题栏一键居中复位。
     - **定点拖动缩放**：右下角专属拉伸抓手，按下后**左、上边框严格固定不动，仅向右、下方向拖动变形**，达到完全原生的桌面窗口体验。
     - **防文字 Tab 变形的最小尺寸限制**：窗口设定了 `min-width: 580px; min-height: 480px`，即使强行向内拉伸也不会压扁内部 Tab 与文字；
     - **自适应填充无留白**：放大时内容区域 `flex: 1 1 100%` 铺满，底部不产生多余无用空白；
     - **双击标题栏全屏最大化 / 一键还原**：增加专属全屏最大化切换按钮与双击标题栏全屏功能；
     - **磨砂毛玻璃通透质感**：背景采用 `backdrop-filter: blur(20px)`，呼应流萤桌面背景。
     - 全局所有使用该弹窗的应用窗口（图库管理、云存储、备份还原、用户管理等）全部立享该交互。

3. **图库管理模块全新重构**：
   - 更名为【图库管理】。
   - 内部拆分为两大分类 Tab：【壁纸图库】与【应用图标】，支持上传时指定分类。
   - **分类互相移动**：支持单张或批量在【壁纸图库 ⇄ 应用图标】之间一键移动切换，按钮图标采用与 Tab 呼应的 `🎨`（移入图标）与 `🌄`（移入壁纸）。
   - **大 Checkbox 与紧凑布局**：选择框放大 1 号（medium 尺寸），减小周围留白让缩略图更大更清晰。
   - **鲜艳状态标识**：【已上云】标签使用高对比度艳红色（`#ff3838`），视觉一目了然。
   - **单击放大再点缩小**：点击图片直接全屏高清预览，再次点击屏幕任意位置或大图本身直接缩小退出，顶部浮动直链与 Markdown 复制胶囊。
   - **工具栏与重命名**：工具栏按钮（全选、移动、推送、删除、清空）字号大 1 号；点击文件名弹出修改框进行重命名。
   - **精简操作栏**：移除原“复制链接”按钮，操作更加清爽。
   - **统一推送图标**：推送到云端图床的图标统一采用系统专属 `cloud.png`，与设为壁纸、删除等按钮图标尺寸保持像素级一致。
   - **批量删除与防误删防呆**：配备多选 Checkbox，支持全选/反选当前页、批量删除选中项、清空全部，并带二次确认危险弹窗。
   - **数据底层防水平越权隔离**：删除接口强制使用 `WHERE user_id = ? AND id IN (?)`，从底层彻底根除普通用户越权删除其他用户或管理员图片的可能。
   - **状态流转与云端推送**：普通用户可对自己的图片点击【申请同步到云端图床】；管理员可直接单张或批量【推送到云端图床】。

4. **云存储与漫游模块升级**：
   - **Tab 标签排序优化**：第 1 页【🖼️ 云端漫游图床】 ➔ 第 2 页【📋 待审图片】 ➔ 第 3 页【⚙️ 存储配置】 ➔ 第 4 页【🔒 云端加密备份】，高频功能优先展现。
   - **云端漫游图床分类与界面精简**：顶部移除原“云端图库与多端漫游”中文字样，仅保留【全部】、【🌄 壁纸图床】、【🎨 应用图床】分段 Tab；右侧刷新、全选当前、清空云端等按钮字号与高度统一对齐（small 尺寸）。
   - **云端资产小仪表盘**：卡片 Header 实时显示云端总张数与占用容量计算。
   - **保守速度限流与彻底解决未删掉 Bug**：
     - 修复了 GitHub API 路径拼接时 BasePath 重复导致查 SHA 404 的隐蔽 Bug，确保远程文件真正被彻底删除；
     - 批量删除时加入 350ms 的保守限流间隔，避免触发 GitHub Secondary Rate Limit 拦截。
   - **管理员审批中心**：一览所有普通用户申请同步的缩略图与申请人，支持【一键全部批准并推送到云端】或单张通过/驳回。

5. **普通用户权限隔离与整站备份防越权**：
   - 普通用户彻底隐藏“整站数据备份与恢复 (ZIP)”Tab，只保留“我的卡片数据 (JSON)”。
   - 后端 `/panel/backup/*` 接口强制加入角色判断 `userInfo.Role != 1` 拦截，杜绝跨权越权。

6. **在线图标“内置图库选择器”弹窗**：
   - 彻底移除跳转外部 `iconify.design` 网页的繁琐操作。
   - 替换为点击弹出【管理员图库选择器】模态框，网格展示公共图库/图标，点击任一图片直接自动填充 URL 并应用为图片图标，无需跳离编辑页面。

---

## 2. 核心架构与模块对应关系

### 前端关键文件
- `src/components/common/RoundCardModal/index.vue`: 左上固定、右下变形的高性能拖拽拉伸模态框容器。
- `src/components/apps/UploadFileManager/index.vue`: 图库管理（分类移动、重命名、批量删除、cloud.png 图标推送、申请同步）。
- `src/components/apps/CloudStorage/index.vue`: 云存储与漫游（Tab 排序、壁纸与图标分类展示、保守限流批量删除、待审图片审批中心）。
- `src/components/apps/ImportExport/index.vue`: 数据导入导出（普通用户权限隔离）。
- `src/components/apps/About/index.vue`: 关于面板（品牌信息与作者 Roland）。
- `src/views/home/components/EditItem/IconEditor.vue`: 卡片图标编辑（内置管理员图库选择器）。
- `src/views/home/components/AppStarter/index.vue`: 系统应用抽屉（菜单顺序与权限渲染）。

### 后端关键文件
- `service/models/file.go`: `File` 模型增加 `FileType`（分类）、`SyncStatus`（流转状态）、`AuditRemark`。
- `service/api/api_v1/system/file.go`:
  - `GetList`: 支持分类与严格用户过滤；
  - `Deletes`: 防水平越权底层强校验；
  - `ChangeCategory`: 移动图片分类；
  - `RenameFile`: 文件重命名；
  - `ApplySyncToCloud`: 用户提交同步申请；
  - `UploadToCloud`: 管理员直接推送云端（分类存放到 wallpapers/icons 目录）；
  - `AuditList`: 管理员待审列表；
  - `AuditAction`: 管理员审批流；
  - `GetPublicIcons`: 公共图标库接口。
- `service/lib/cloudClient/client.go`: `CloudStorageClient` 接口扩展 `DeleteFile`，修复路径拼接，确保 GitHub 文件真删除。
- `service/api/api_v1/panel/cloudStorage.go`: 批量删除（保守限流）、清空图床、多目录合并列出图片接口。
- `service/api/api_v1/panel/backup.go`: 整站全量备份与还原接口强权限校验。

---

## 3. 构建与部署说明

### 前端构建
```bash
cd source
pnpm run build-only
```

### 后端构建
```bash
cd source/service
go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release" main.go
```

### 多架构 Docker 镜像
基于 GitHub Actions 自动构建并推送到 GitHub Container Registry (GHCR)：
- 镜像仓库: `ghcr.io/mysecret2022/my-sun-panel:1.0.2` & `latest`
- 架构支持: `linux/amd64`, `linux/arm64`

---

## 4. 架构与生态联动 (Ecosystem Integration)
- **项目与仓库正式统一命名**：已正式统一更名为 **`firefly-panel`**。
- **全套配套联动状态**：
  - [x] GitHub 远程仓库重命名为 `mysecret2022/firefly-panel` (https://github.com/mysecret2022/firefly-panel)
  - [x] GitHub Remote URL 更新为 `https://github.com/mysecret2022/firefly-panel.git`
  - [x] GitHub Actions 镜像发布路径自动对应 `ghcr.io/mysecret2022/firefly-panel`
  - [x] 增加专属 `:dev` 测试版镜像标签，并保留 `:latest` 和版本号标签
  - [x] `docker-compose.yml` 中的服务名与镜像名更新为 `ghcr.io/mysecret2022/firefly-panel:dev`
  - [x] `package.json` 中的 `name` 规范为 `firefly-panel`
  - [x] `README.md` 与文档中所有相关引用全面对齐。

