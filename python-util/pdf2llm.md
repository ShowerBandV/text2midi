以下是使用 **PyMuPDF4LLM** 将 PDF 教材转换为 Markdown 文本的完整代码示例，适合你批量处理音乐理论书籍并导入 AI 知识库。

### 1. 安装

```bash
pip install pymupdf4llm
```

### 2. 单文件转换（基础用法）

```python
import pymupdf4llm

# 将 PDF 转为 Markdown 文本
md_text = pymupdf4llm.to_markdown("和声学教程.pdf")

# 保存为 .md 文件
import pathlib
pathlib.Path("和声学教程.md").write_bytes(md_text.encode())
print("转换完成")
```

### 3. 批量转换并保留图片（适合有谱例的乐理书）

```python
import pymupdf4llm
import pathlib

pdf_dir = pathlib.Path("./pdfs")
output_dir = pathlib.Path("./md_output")
output_dir.mkdir(exist_ok=True)

for pdf_file in pdf_dir.glob("*.pdf"):
    print(f"正在处理: {pdf_file.name}")
    try:
        md_text = pymupdf4llm.to_markdown(
            pdf_file,
            write_images=True,                # 提取图片到本地
            image_path="images",               # 图片保存目录
            image_format="png",                # 图片格式
            dpi=200                            # 图片分辨率（提高清晰度）
        )
        output_path = output_dir / f"{pdf_file.stem}.md"
        output_path.write_bytes(md_text.encode())
    except Exception as e:
        print(f"转换失败 {pdf_file.name}: {e}")

print("全部转换完成")
```

### 4. 高级选项：按页输出、控制表格、忽略错误

```python
import pymupdf4llm

md_text = pymupdf4llm.to_markdown(
    "配器法教程.pdf",
    pages=[0, 5, 10],            # 只转换特定页码（0-based）
    page_chunks=True,            # 返回按页分割的 list[dict]
    write_images=True,
    image_path="orchestration_images",
    image_format="jpeg",
    dpi=150,
    table_strategy="lines",      # 表格识别策略（lines/text/auto）
    ignore_errors=False          # True 则跳过无法渲染的图形
)

# 如果 page_chunks=True，返回格式如：
# [{"metadata": {...}, "text": "第一页内容..."}, {"metadata": {...}, "text": "第二页内容..."}]
```

### 5. 直接用于 LLM 上下文（链式调用）

```python
import pymupdf4llm

prompt = """
你是一名音乐理论家，请根据以下教材内容回答学生的问题。

教材内容：
{text}

问题：如何理解副属和弦的功能？
"""

md_text = pymupdf4llm.to_markdown("和声学教程.pdf", pages=[34,35])
full_prompt = prompt.format(text=md_text)

# 之后发送到 DeepSeek API 等
# response = client.chat.completions.create(...)
```

### ⚠️ 注意事项

- **乐谱图片**：PyMuPDF4LLM 会将图表转为单独图片并在 Markdown 中插入 `![]()` 引用，但图片本身不会被 LLM“看懂”。如果你需要用 AI 分析谱例，建议配合多模态模型（如 GPT-4o）或先使用专门的乐谱 OCR 工具（如 Audiveris）。
- **中文书籍**：PyMuPDF4LLM 内部使用 PyMuPDF，对中文 PDF 的文本提取较好，但部分扫描版可能乱码，需要先做 OCR 预处理（如使用 OCRmyPDF）。
- **文件大小**：若生成大量图片，请注意磁盘占用；可以适当降低 dpi 或仅在需要时提取图片。

如果需要自动化处理整个书籍目录，我可以帮你写一个更完整的流水线脚本（包括文件组织、错误重试和日志）。