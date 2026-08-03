import { getToken } from './client';

// ─── Reports ───
// Downloads a session report in the given format via a direct browser download
// (full URL built from the current host so the auth token header is attached).

export type ReportFormat = 'html' | 'pdf' | 'docx' | 'xlsx';

export function exportReport(sessionId: string, format: ReportFormat) {
  // Use a fetch to include the Authorization header, then trigger a download
  // from the returned blob.
  const token = getToken();
  const headers: Record<string, string> = {};
  if (token) headers['Authorization'] = `Bearer ${token}`;

  fetch(`/api/v1/sessions/${sessionId}/report?format=${format}`, { headers })
    .then((res) => {
      if (!res.ok) {
        throw new Error(`导出失败 (HTTP ${res.status})`);
      }
      return res.blob();
    })
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const stamp = new Date().toISOString().slice(0, 10);
      a.download = `aster-report-${sessionId.slice(0, 8)}-${stamp}.${format}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    })
    .catch((err) => {
      // Re-throw so the calling UI can surface the error.
      throw err;
    });
}
