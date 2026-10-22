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

---

## 5. 本轮功能迭代与体验全面升级 (v1.0.2-dev Update)
已全量落地并完成前后端双重编译验证：

1. **登录界面 (Login View) 视觉与交互全面升级**：
   - [x] **登录背景图**：替换为专属流萤壁纸 `firefly.webp`，自适应全屏铺满。
   - [x] **通透毛玻璃卡片**：登录框升级为苹果 iOS 旗舰级毛玻璃材质（`backdrop-filter: blur(24px) saturate(180%)`，带透光微边框与立体柔光投影）。
   - [x] **密码明密文眼睛图标**：密码输入框增加原生眼睛图标，点击可自由切换查看明文与掩码密文（`show-password-on="click"`）。
   - [x] **系统标识更新**：登录框上方的标题统一规范为 **`firefly-panel`**。
   - [x] **占位提示纯净交互**：输入框内的“账号/密码”占位符深度优化，获得焦点与敲字时彻底物理隐形 Naive UI 内部占位符 span，输入法拼音输入亦绝对零遮挡无残留。

2. **主页 Header 标题栏与 Footer 页脚 (苹果 iOS 27 超感毛玻璃质感)**：
   - [x] **透光毛玻璃文字设计 (Glassmorphic Text)**：主页顶部的 `Firefly-panel` 标题、优雅细分割线与时间/日期**不使用封闭药丸外框**，而是让文字本身呈现高科技毛玻璃通透折射质感（半透微光折射、壁纸若隐若现透出、高透柔光微阴影与液态通透感），视觉极度轻盈。
   - [x] **页脚 Powered By iOS 27 毛玻璃升级**：主页底部的 `Powered By Firefly-panel` 专属段落全面升级为 iOS 27 超感毛玻璃悬浮透光胶囊，仓库地址自动对齐到最新 `firefly-panel`。

3. **云存储与漫游模块 (交互痛点与布局修复)**：
   - [x] **Header 按钮与已漫游同一行排布**：【已漫游：X 张 容量：Y 云端畅通】与【刷新】、【全选当前】、【批量删除】、【清空云端】、【一键全部拉取到本地】全部整合并排在同一行，布局极为整洁规整。
   - [x] **云端图片弹性平铺一排**：网格断点升级为弹性自适应 `cols="2 450:3 680:4 900:6 1200:8 1500:10"`，向右拉伸时自动平铺成一排排，并撑满高度消除底部死角空白。
   - [x] **规范上云命名与去除遮挡**：
     - 上云时按【原始文件名 + 6位短校验码】规范命名（如 `Chrome_a1b2c3.png`）；
     - 彻底移除盖在图片上方的【壁纸/图标】字体 Tag，缩略图纯净无遮挡。
   - [x] **云端加密备份历史支持手动删除与本地下载**：增加红色【删除】按钮（带二次确认防误触弹窗）与【下载】按钮，支持手动清理冗余备份包。
   - [x] **云端图片全屏大图预览**：单击任意图片全屏放大高清预览，再点击大图或任意位置秒退缩小。

4. **图库管理与系统弹窗自适应优化**：
   - [x] **图库管理全屏查看器回归原生 NImage**：删除自定义预览浮层上的【复制链接】和【复制 Markdown】标签，彻底恢复原版 Naive UI 原生全屏大图查看器（内置滚轮缩放、放大/缩小/旋转/100%还原工具栏、鼠标拖拽移动，完整无截断展示长图宽图），保留每张卡片底部的【删除此图片】功能。
   - [x] **图库图标动态可调列数**：突破原先最多 6 个的死锁限制，支持自适应扩展至 12 列，并提供专属排布密度调节器（自适应/4/6/8/10/12列）。
   - [x] **放大弹窗底部留白彻底根除**：底层重构 `RoundCardModal` 的插槽约束与子组件 flex 链条，强制各级卡片和滚动区 100% 充满全部可用高度，告别底部空洞留白。

5. **数据导出导入模块 (ImportExport)**：
   - [x] **普通用户直接展示单账号导入导出内容**：修复 `activeTab` 默认值与 TabPane 名称 `json-cards` 不匹配导致的空白问题（之前初始化为未定义的 `card-backup`）。普通账户打开该模块直接展示“单账号卡片导入导出 (JSON)”内容，彻底消除“需要点击 Tab 才能展示”的问题。

