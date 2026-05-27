"""
PDF -> Markdown batch conversion v5 — single-thread per book, multi-PDF via subprocess
Based on pdf2llm.md prompt integration + PaddleOCR

Strategy: PaddlePaddle is CPU-bound internally, so threading doesn't help.
This version uses multiprocessing to convert different PDFs in parallel.

Usage:
  python batch_pdf2md.py                          # auto, one-by-one
  python batch_pdf2md.py --parallel 2             # convert 2 PDFs at once
  python batch_pdf2md.py --parallel               # convert all at once
  python batch_pdf2md.py --check                  # PDF type only
  python batch_pdf2md.py --fast                   # pymupdf4llm only
"""

import pathlib
import sys
import time
import argparse
import subprocess
from datetime import datetime

# ======== Config =========
PDF_DIR = pathlib.Path(r"E:\game_demos\text2midi\music_wiki_raw")
OUTPUT_DIR = PDF_DIR

WRITE_IMAGES = True
DPI = 150

OCR_LANG = "ch"
OCR_CONFIDENCE_THRESHOLD = 0.5
TEXT_PAGE_THRESHOLD = 50
# =========================


def get_pdf_info(pdf_path: pathlib.Path) -> tuple[str, int]:
    import fitz
    doc = fitz.open(str(pdf_path))
    total = len(doc)
    text_pages = sum(1 for page in doc if len(page.get_text().strip()) > TEXT_PAGE_THRESHOLD)
    doc.close()
    if text_pages > total * 0.5:
        desc = f"text({text_pages}/{total})"
    elif text_pages > 0:
        desc = f"mixed({text_pages}/{total})"
    else:
        desc = f"scanned(0/{total})"
    return desc, total


def convert_one_pdf(pdf_path: pathlib.Path, output_path: pathlib.Path):
    """Convert a single PDF (called in subprocess)."""
    import fitz
    from PIL import Image
    import numpy as np
    from paddleocr import PaddleOCR

    desc, total = get_pdf_info(pdf_path)
    is_scanned = desc.startswith("scanned") or desc.startswith("mixed")

    if not is_scanned:
        # Text PDF — use pymupdf4llm
        import pymupdf4llm
        print(f"[{pdf_path.name}] Text PDF, using pymupdf4llm...")
        md_text = pymupdf4llm.to_markdown(
            str(pdf_path), write_images=WRITE_IMAGES,
            image_path=str(OUTPUT_DIR / pdf_path.stem / "images"),
            image_format="png", dpi=DPI, ignore_errors=True)
        md_text = f"---\nsource: {pdf_path.name}\nconverted: {datetime.now().isoformat()}\nmethod: pymupdf4llm\n---\n\n{md_text}"
        output_path.write_bytes(md_text.encode("utf-8-sig"))
        print(f"[{pdf_path.name}] Done: {output_path.stat().st_size/1024:.1f} KB")
        return

    # Scanned PDF — use PaddleOCR
    print(f"[{pdf_path.name}] Scanned ({total}p), OCR starting...")
    ocr = PaddleOCR(lang=OCR_LANG, ocr_version="PP-OCRv4")

    doc = fitz.open(str(pdf_path))
    md_pages = []
    start_time = time.time()

    for i in range(total):
        t0 = time.time()
        page = doc[i]
        mat = fitz.Matrix(DPI / 72, DPI / 72)
        pix = page.get_pixmap(matrix=mat, alpha=False)
        img = Image.frombytes("RGB", [pix.width, pix.height], pix.samples)
        img_arr = np.array(img)

        result = ocr.ocr(img_arr)
        texts = []
        if result and result[0]:
            if isinstance(result[0], dict):
                for t, c in zip(result[0].get('rec_texts', []), result[0].get('rec_scores', [])):
                    t = t.strip()
                    if t and c >= OCR_CONFIDENCE_THRESHOLD:
                        texts.append(t)
            else:
                for line in result[0]:
                    t, c = line[1][0], line[1][1]
                    if c >= OCR_CONFIDENCE_THRESHOLD:
                        texts.append(t)

        md_pages.append("\n\n".join(texts) if texts else f"*[Page {i+1}: empty]*")

        elapsed = time.time() - t0
        avg = (time.time() - start_time) / (i + 1)
        eta = (total - i - 1) * avg
        pct = (i + 1) / total * 100
        print(f"  [{pdf_path.name}] {i+1}/{total} ({pct:.0f}%) {elapsed:.1f}s ETA {eta:.0f}s")

    doc.close()

    final = f"---\nsource: {pdf_path.name}\nconverted: {datetime.now().isoformat()}\npages: {total}\nmethod: OCR\n---\n\n"
    final += "\n\n---\n\n".join(md_pages)
    output_path.write_bytes(final.encode("utf-8-sig"))

    total_time = time.time() - start_time
    size = output_path.stat().st_size
    print(f"[{pdf_path.name}] Done: {size/1024:.1f} KB, {total_time:.0f}s ({total_time/total:.1f}s/page)")


def main():
    parser = argparse.ArgumentParser(description="PDF to Markdown batch converter")
    parser.add_argument("--check", action="store_true")
    parser.add_argument("--fast", action="store_true", help="pymupdf4llm only")
    parser.add_argument("--parallel", type=int, nargs='?', const=0, default=1,
                        help="convert N PDFs in parallel (default: 1, --parallel: all)")
    parser.add_argument("--pdf-dir", type=str, default=str(PDF_DIR))
    args = parser.parse_args()

    pdf_dir = pathlib.Path(args.pdf_dir)
    if not pdf_dir.is_dir():
        print(f"Error: {pdf_dir} not found"); sys.exit(1)

    pdf_files = sorted(pdf_dir.glob("*.pdf")) or sorted(pdf_dir.glob("*.PDF"))
    if not pdf_files:
        print("No PDFs found"); sys.exit(0)

    print(f"Source: {pdf_dir}  |  {len(pdf_files)} PDFs  |  Parallel: {args.parallel}")

    if args.check:
        for pdf in pdf_files:
            desc, total = get_pdf_info(pdf)
            print(f"  [{desc:20s}] {pdf.name[:60]}")
        return

    # Filter already-done
    todo = []
    for pdf in pdf_files:
        out = OUTPUT_DIR / f"{pdf.stem}.md"
        if out.exists() and out.stat().st_size > 0:
            print(f"  SKIP: {pdf.name} -> exists ({out.stat().st_size/1024:.1f} KB)")
        else:
            todo.append(pdf)

    if not todo:
        print("All done."); return

    if args.parallel <= 1:
        # Sequential
        for pdf in todo:
            out = OUTPUT_DIR / f"{pdf.stem}.md"
            convert_one_pdf(pdf, out)
    else:
        # Parallel: spawn subprocess for each
        max_workers = args.parallel if args.parallel > 0 else len(todo)
        import multiprocessing
        ctx = multiprocessing.get_context('spawn')
        with ctx.Pool(max_workers) as pool:
            tasks = [(pdf, OUTPUT_DIR / f"{pdf.stem}.md") for pdf in todo]
            pool.starmap(convert_one_pdf, tasks)

    print(f"\nAll done! Output: {OUTPUT_DIR}")


if __name__ == "__main__":
    main()
