"""Quick PaddleOCR test"""
import fitz, os, numpy as np
os.environ['PADDLE_PDX_DISABLE_MODEL_SOURCE_CHECK'] = 'True'
from PIL import Image
from paddleocr import PaddleOCR

pdf = r'E:\game_demos\text2midi\music_wiki_raw\伯克利流行歌曲写作 _ 和声 (Berklee College of Music) (z-library.sk, 1lib.sk, z-lib.sk).pdf'
doc = fitz.open(pdf)
print(f'Total: {len(doc)} pages')

ocr = PaddleOCR(lang='ch', ocr_version='PP-OCRv4')

for i in range(2):
    page = doc[i]
    mat = fitz.Matrix(150/72, 150/72)
    pix = page.get_pixmap(matrix=mat, alpha=False)
    img = Image.frombytes('RGB', [pix.width, pix.height], pix.samples)
    img_arr = np.array(img)

    result = ocr.ocr(img_arr)
    if result and result[0] and isinstance(result[0], dict):
        texts = result[0].get('rec_texts', [])
        scores = result[0].get('rec_scores', [])
        print(f'\n--- Page {i+1} ({len(texts)} items) ---')
        for t, c in zip(texts, scores):
            t = t.strip()
            if t and c >= 0.3:
                print(f'  [{c:.2f}] {t}')
    elif result and result[0]:
        print(f'\n--- Page {i+1} (legacy format) ---')
        for line in result[0]:
            text = line[1][0]
            conf = line[1][1]
            if conf >= 0.3:
                print(f'  [{conf:.2f}] {text}')
    else:
        print(f'\n--- Page {i+1}: no text ---')

doc.close()
print('\nDone')
