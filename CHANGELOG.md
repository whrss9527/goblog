# Changelog

本项目的所有重要变更都记录在这里。版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)，
当前版本同时写在 `internal/version/version.go` 中（页面 `<meta name="generator">` 会带上它）。

## [1.1.0] - 2026-09-19 · 修复与地基

这一版先把主干上几个「看不见但很伤」的问题修掉，并为后续的体验迭代打地基。

### 修复
- **文章 / 页面正文空白**：`editormd.js` 在 jQuery 之前加载，抛出 `Zepto is not defined`，
  正文永远渲染不出来。Markdown 相关脚本现在统一放进 layout 的 `scripts` 块（jQuery 之后）。
- **正文等所有资源加载完才出现**：渲染不再挂在 `window.onload` 上（会被图片、giscus、统计脚本拖住），
  脚本就绪后立即渲染。
- **标签页热力图不显示**：`html/template` 把 JSON 又包成了一层 JS 字符串，`JSON.parse` 得到的是字符串，
  `rawData.forEach is not a function`。改为以 `template.JS` 输出；顺带修复年份下拉框默认选中项与内容不一致。
- **阅读清单封面 404**：`books.json` 里的 `/covers/*.jpg` 没有对应路由，现在由 `data_dir/covers` 提供。
- **图标全部是空白方块**：`/static/fonts/glyphicons-*` 不存在。改用内联 SVG 图标（`tpl/default/icons.html`），
  不再依赖 Glyphicons / FontAwesome 字体。
- **搜索结果翻页丢关键词**：分页链接现在保留 `keyword`，并改为站内相对地址（不再拼 `app.host`）。
- **软 404**：不存在的文章 / 页面返回真正的 404 状态码并带 `noindex`；未知路由也走主题化的 404 页面
  （之前是 gin 的纯文本 `404 page not found`）。
- 非 Apple 设备上标题被渲染成宋体：去掉内联的 `PingFangSC-Light, serif`，改为带中文回退的系统字体栈。

### 优化
- **暗色模式不再闪白**：主题在 `<head>` 内联脚本里于首次绘制前生效；没有手动选择过时跟随系统
  `prefers-color-scheme`，并实时响应系统切换。主题开关支持键盘操作。
- 去掉 `<head>` 里阻塞渲染的第三方 quicklink 脚本，改为内置的「悬停 / 触摸预取」（只预取用户即将打开的链接，
  而不是视口内所有链接；省流量模式与 2G 下自动关闭）。
- 移除前台从未生效的 Cloudflare Turnstile 脚本（目标容器 `#example-container` 不存在，只会报错）。
- flowchart / sequence-diagram / raphael / underscore / KaTeX 改为按需加载：只有文章里真的用到
  <code>```flow</code>、<code>```seq</code>、`$$` 时才会请求（目前 44 篇文章都用不到，每次阅读少加载 ~200KB JS）。
- giscus 评论改为滚动到附近时才加载（`IntersectionObserver`），主题与站点明暗实时同步。
- 本地静态资源带内容指纹（`?v=<sha1 前 10 位>`，模板函数 `asset`），发版后浏览器 / CDN 缓存立即失效。
- 模板先渲染到缓冲区再输出，模板出错时返回 500 而不是半截页面。
- 代码块「复制」按钮：兼容非安全上下文，修复 prettify 行号模式下复制内容丢换行的问题。
- 头像仍然通往 `/intro`（保留彩蛋），站点名称链接到首页；页脚年份自动更新。

### 工程
- 新增 `internal/version`、`CHANGELOG.md`；新增 `getPageUrl`、`AssetURL` 单元测试。
- `conf/prod.yaml.example` 中的 `session_secret` 换成占位符（原值看起来像真实密钥，**如果线上在用请尽快轮换**）。
