import { get, post } from '@/utils/request'
import { useAuthStore } from '@/store'

export function exportBackup() {
  const authStore = useAuthStore()
  const token = authStore.token
  const baseUrl = import.meta.env.VITE_GLOB_API_URL || '/api/v1'
  const url = `${baseUrl}/panel/backup/export?token=${token || ''}`

  // 格式化当前本地日期与时间：YYYYMMDD_HHmmss
  const now = new Date()
  const pad = (n: number) => n.toString().padStart(2, '0')
  const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`

  // 触发带有完整日期和时间的 ZIP 文件下载
  const a = document.createElement('a')
  a.href = url
  a.download = `sun-panel-backup_${timestamp}.zip`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

export function exportJson<T>() {
  return get<T>({
    url: '/panel/backup/exportJson',
  })
}

export function restoreBackup<T>(formData: FormData) {
  return post<T>({
    url: '/panel/backup/restore',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  })
}
