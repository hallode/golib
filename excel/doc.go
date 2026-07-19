// Package excel provides helpers over excelize/v2 for generating XLSX files.
//
// Pick the entry point that fits the workload:
//
//   - GenerateExcel — a small, fully in-memory single sheet from headers + rows.
//   - StreamSheet (NewStreamSheet) — one large sheet written row-by-row with
//     bounded memory; NewStreamSheetFromLayout adds merged header rows.
//   - MultiStreamWorkbook — several streamed sheets in one workbook.
//   - GenerateDynamicExcelStreaming — a JSON-driven export that pulls rows in
//     batches via a callback.
//
// Streaming writers hold an open underlying file: always finish with Bytes
// (which flushes) or Close to avoid leaking resources.
package excel
