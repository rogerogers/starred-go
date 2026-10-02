# starred (Go 语言重写版)

> Creating your own Awesome List used GitHub stars! (Rewritten in pure Go)

基于 [maguowei/starred](https://github.com/maguowei/starred) 的完整 **Go 语言重写版**。

## 🌟 特性优势

- ⚡️ **零外部依赖（Zero External Dependencies）**：基于 Go 标准库（`net/http`、`encoding/json`、`flag`）实现，无需引入任何第三方重型库。
- 📦 **单二进制交付**：无需预装 Python 解释器与 virtualenv，编译后生成开箱即用的轻量可执行文件。
- 🚀 **极致性能**：极低的内存占用与并发网络优化，毫秒级冷启动。
- 🎯 **100% 格式对齐**：完全兼容原版 Python 版的 Markdown 排版规范、HTML 转义规则、锚点链接算法、Category 标题转义与 License 声明。
- 🔄 **支持直接提交到 GitHub**：自动检测/创建目标仓库并推送 Commit。

---

## 🛠️ 编译与安装

确保本地已安装 Go（1.21 或更高版本）：

```bash
cd /Users/rogers/.gemini/antigravity/scratch/starred-go

# 编译为本地二进制可执行文件
go build -o starred .

# 或者直接全局安装到 $GOPATH/bin
go install .
```

---

## 📖 命令行使用说明

```text
starred (Go rewrite) v4.3.0-go

Usage:
  starred [options]

Options:
  -u, --username      GitHub 用户名 [也可读取环境变量 USER 或 GITHUB_USER]
  -t, --token         GitHub Personal Access Token [也可读取环境变量 GITHUB_TOKEN]
      --sort          按分类（语言或标签）名称字母顺序排序 (默认: 保持 star 先后顺序)
      --topic         按项目主题（Topic）分类 (默认: 按主编程语言分类)
      --topic-limit   Topic 筛选门槛，仅统计 star 数大于该值的 topic (默认: 500)
  -r, --repository    自动提交到的目标 GitHub 仓库名 (如 awesome-stars)
  -f, --filename      目标文件名 (默认: README.md)
  -m, --message       Git 提交信息 (默认: "update awesome-stars, created by starred-go")
      --private       包含私有仓库 (默认: false)
  -o, --out           直接输出到本地文件路径
  -v, --version       查看版本号
  -h, --help          查看帮助信息
```

---

## 💡 使用示例

### 1. 导出到本地 README.md

```bash
# 通过管道重定向
./starred --username your_github_name --token=ghp_xxxx --sort > README.md

# 或使用 -o 参数直接写出文件
./starred -u your_github_name -t ghp_xxxx --sort -o README.md
```

### 2. 按 Topic 分类排序

```bash
./starred -u your_github_name -t ghp_xxxx --topic --topic-limit 1000 -o README.md
```

### 3. 直接同步并提交到 GitHub 仓库

```bash
./starred -u your_github_name -t ghp_xxxx --repository awesome-stars --sort
```
*(如果 `awesome-stars` 仓库不存在，程序会自动通过 GitHub REST API 为你创建新仓库并提交 `README.md`)*

---

## 🤖 GitHub Actions 自动化示例

你可以在 GitHub Actions 中用 Go 编译并每天定时更新自己的 Awesome Stars：

```yaml
name: Update Awesome Stars

on:
  schedule:
    - cron: '0 0 * * *' # 每天凌晨更新
  workflow_dispatch:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run starred
        env:
          GITHUB_TOKEN: ${{ secrets.ACCESS_TOKEN }}
        run: |
          go run ./... --username ${{ github.repository_owner }} --repository ${{ github.event.repository.name }} --sort
```

---

## 🧪 运行单元测试

```bash
go test -v ./...
```
