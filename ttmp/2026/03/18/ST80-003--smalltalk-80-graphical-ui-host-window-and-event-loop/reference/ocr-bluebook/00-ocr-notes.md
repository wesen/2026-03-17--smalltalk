# OCR Notes — Blue Book Extraction

## Source
- **Book:** *Smalltalk-80: The Language and its Implementation* by Adele Goldberg and David Robson (1983)
- **PDF:** `smalltalk-Bluebook.pdf` (742 pages, 33MB, PDF 1.4)
- **Page offset:** PDF page = printed page + 22 (e.g., printed p.329 = PDF p.351)

## OCR Approach
- Primary tool: `pdftotext` (poppler-utils 24.02.0) for bulk text extraction
- Visual verification: Claude Code PDF reader for direct page images
- Cross-check: Every extracted fact verified against visual page scan

## Tools Used
- `pdftotext` — bulk text extraction per chapter
- Claude Code built-in PDF reader — visual verification of class definitions, tables, figures
- No OCR engine (tesseract/ocrmypdf) was needed; the PDF has embedded text layer

## Scan Quality Issues
- Text layer is generally good quality; most pages read cleanly
- Chapter opening pages have decorative graphics that produce garbage in text extraction
- Smalltalk code examples: arrow characters (←) sometimes render as other characters
- Some pages have OCR artifacts from the original scan (dots, noise in margins)
- Figures (e.g., 27.1-27.8, 30.1-30.8) are image-only; data was read from visual scans

## Pages Needing Manual Correction
- pp. 338-340 (Form/Bitmap class definitions): class definition layout verified visually
- pp. 349-351 (BitBlt parameter table): verified visually against text extraction
- pp. 355-362 (BitBltSimulation code): code verified line by line from visual scan
- pp. 575-576 (guaranteed pointers): pointer values verified visually
- pp. 577-579 (method header bit fields): bit layout verified from Figure 27.2-27.4
- pp. 590-591 (instance specification): bit layout verified from Figure 27.8
- pp. 612-615 (primitive table): entire table verified visually
- pp. 661-662 (object table entry layout): bit fields verified from Figure 30.5

## Unresolved OCR Ambiguities
- None significant. The PDF text layer is high quality and all critical values were confirmed by visual inspection.

## Raw Text Extractions
The following raw text files were saved for future reference:
- `raw-ch18-graphics-kernel.txt` — Chapter 18, pp. 329-368
- `raw-ch20-display-objects.txt` — Chapter 20, pp. 381-398
- `raw-ch27-vm-spec.txt` — Chapter 27, pp. 567-592
- `raw-ch29-primitives.txt` — Chapter 29, pp. 611-654
- `raw-ch30-object-memory.txt` — Chapter 30, pp. 655-690
