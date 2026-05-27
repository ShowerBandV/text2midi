"""Convert ONE PDF to Markdown (with periodic checkpoint saving)."""
import sys, time, pathlib
from datetime import datetime

CHECKPOINT_INTERVAL = 10  # save every N pages

def convert(pdf_path: str, output_path: str):
    pdf = pathlib.Path(pdf_path)
    out = pathlib.Path(output_path)

    import fitz
    doc = fitz.open(str(pdf))
    total = len(doc)
    text_pages = sum(1 for p in doc if len(p.get_text().strip()) > 50)
    is_text = text_pages > total * 0.5
    print(f"[{pdf.name}] {total}p, text={text_pages}, method={'pymupdf4llm' if is_text else 'PaddleOCR'}")
    sys.stdout.flush()

    if is_text:
        import pymupdf4llm
        md = pymupdf4llm.to_markdown(str(pdf), write_images=True, dpi=150, ignore_errors=True)
        md = f"---\nsource: {pdf.name}\nconverted: {datetime.now().isoformat()}\nmethod: pymupdf4llm\n---\n\n{md}"
        out.write_bytes(md.encode("utf-8-sig"))
        print(f"[{pdf.name}] DONE: {out.stat().st_size/1024:.1f} KB")
        return

    from PIL import Image
    import numpy as np
    from paddleocr import PaddleOCR

    ocr = PaddleOCR(lang='ch', ocr_version='PP-OCRv4')
    doc = fitz.open(str(pdf))
    results = []
    t_start = time.time()

    def save_checkpoint():
        """Save partial results so far."""
        md = f"---\nsource: {pdf.name}\nconverted: {datetime.now().isoformat()}\npages: {total}\nmethod: OCR(checkpoint)\n---\n\n"
        md += "\n\n---\n\n".join(results)
        if results:
            md += f"\n\n*--- CHECKPOINT: {len(results)}/{total} pages ---*"
        out.write_bytes(md.encode("utf-8-sig"))
        size_kb = out.stat().st_size / 1024
        print(f"[{pdf.name}] CHECKPOINT saved: {len(results)}/{total}p, {size_kb:.1f} KB")

    for i in range(total):
        t0 = time.time()
        page = doc[i]
        mat = fitz.Matrix(150/72, 150/72)
        pix = page.get_pixmap(matrix=mat, alpha=False)
        img = Image.frombytes("RGB", [pix.width, pix.height], pix.samples)
        arr = np.array(img)

        res = ocr.ocr(arr)
        texts = []
        if res and res[0]:
            if isinstance(res[0], dict):
                for t, c in zip(res[0].get('rec_texts',[]), res[0].get('rec_scores',[])):
                    t = t.strip()
                    if t and c >= 0.5: texts.append(t)
            else:
                for ln in res[0]:
                    t, c = ln[1][0], ln[1][1]
                    if c >= 0.5: texts.append(t)

        results.append("\n\n".join(texts) if texts else f"*[{i+1}]*")

        elapsed = time.time() - t_start
        avg = elapsed / (i+1)
        eta = (total - i - 1) * avg
        print(f"[{pdf.name}] P{i+1}/{total} ({time.time()-t0:.1f}s, avg{avg:.1f}s, ETA{eta:.0f}s)")
        sys.stdout.flush()

        # Checkpoint every N pages
        if (i + 1) % CHECKPOINT_INTERVAL == 0:
            save_checkpoint()

    doc.close()

    # Final write (overwrite checkpoint with complete version)
    md = f"---\nsource: {pdf.name}\nconverted: {datetime.now().isoformat()}\npages: {total}\nmethod: OCR\n---\n\n"
    md += "\n\n---\n\n".join(results)
    out.write_bytes(md.encode("utf-8-sig"))
    print(f"[{pdf.name}] DONE: {out.stat().st_size/1024:.1f} KB, {time.time()-t_start:.0f}s")

if __name__ == "__main__":
    convert(sys.argv[1], sys.argv[2])
