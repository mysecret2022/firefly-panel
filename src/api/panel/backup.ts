import { get, post } from '@/utils/request'
import { useAuthStore } from '@/store'

export function exportBackup() {
  const authStore = useAuthStore()
  const token = authStore.token
  const baseUrl = import.meta.env.VITE_GLOB_API_URL || '/api/v1'
  const url = `${baseUrl}/panel/backup/export?token=${token || ''}`
  
  // 使用隐藏 a 标签或 iframe 触发原生附件流下载
  const a = document.createElement('a')
  a.href = url
  a.download = `sun-panel-backup_${new Date().toISOString().slice(0, 10)}.zip`
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
