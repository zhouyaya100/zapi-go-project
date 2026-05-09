import api from './index'

function download(url: string, filename: string) {
  return api.get(url, { responseType: 'blob' }).then(r => {
    const blob = new Blob([r.data])
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = filename
    link.click()
    URL.revokeObjectURL(link.href)
  })
}

export const exportApi = {
  adminCsv(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/export${q}`, 'report.csv')
  },
  adminXlsx(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/export/xlsx${q}`, 'report.xlsx')
  },
  operatorCsv(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/export/operator${q}`, 'report.csv')
  },
  operatorXlsx(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/export/operator/xlsx${q}`, 'report.xlsx')
  },
  myCsv(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/my/export${q}`, 'my_report.csv')
  },
  myXlsx(params?: Record<string, unknown>): Promise<void> {
    const q = params ? '?' + new URLSearchParams(params as Record<string, string>).toString() : ''
    return download(`/api/reports/my/export/xlsx${q}`, 'my_report.xlsx')
  },
}
