import * as XLSX from 'xlsx';

/**
 * Converts a spreadsheet to CSV in the browser, so the import flow only ever
 * has to understand one format.
 *
 * The server parses CSV and nothing else. Teaching it spreadsheets would mean a
 * second parser, a second set of edge cases and a second thing to keep in step
 * with the first; converting here means header extraction, column mapping,
 * preview and de-duplication carry on working exactly as they already do,
 * against a file the server already knows how to read.
 *
 * Only the first worksheet is read. A workbook with several sheets almost
 * always keeps the contacts on one of them, and silently concatenating the rest
 * would import whatever notes or lookup tables happen to sit alongside.
 */
export const SPREADSHEET_EXTENSIONS = ['.xlsx', '.xls'];

export function isSpreadsheet(file: File): boolean {
  return SPREADSHEET_EXTENSIONS.some((ext) =>
    file.name.toLowerCase().endsWith(ext)
  );
}

export async function spreadsheetToCsvFile(file: File): Promise<File> {
  const buffer = await file.arrayBuffer();
  // cellDates keeps real dates as dates; without it they arrive as Excel's
  // serial numbers and a "last contacted" column imports as 45312.
  const book = XLSX.read(buffer, { type: 'array', cellDates: true });

  const sheetName = book.SheetNames[0];
  if (!sheetName) {
    throw new Error('That workbook has no sheets.');
  }
  const sheet = book.Sheets[sheetName];

  // blankrows: false drops the empty rows spreadsheets accumulate below real
  // data, which would otherwise import as blank contacts.
  const csv = XLSX.utils.sheet_to_csv(sheet, { blankrows: false }).trim();
  if (!csv) {
    throw new Error(`The first sheet ("${sheetName}") is empty.`);
  }

  const csvName = file.name.replace(/\.(xlsx|xls)$/i, '') + '.csv';
  return new File([csv], csvName, { type: 'text/csv' });
}

/** Passes CSV through untouched; converts a spreadsheet. */
export async function toCsvFile(file: File): Promise<File> {
  return isSpreadsheet(file) ? spreadsheetToCsvFile(file) : file;
}
