import { get, post } from '@/utils/request'

export interface FileItem {
  id: number
  src: string
  fileName: string
  fileType: number // 0: 壁纸/图库, 1: 应用图标
  syncStatus: number // 0: 仅本地, 1: 待审核, 2: 已同步云端, 3: 已驳回
  userId?: number
  userName?: string
  createTime: string
  updateTime?: string
  path: string
}

export function getList<T>(params?: { fileType?: number }) {
  return post<T>({
    url: '/file/getList',
    params,
  })
}

export function deletes<T>(ids: number[]) {
  return post<T>({
    url: '/file/deletes',
    data: { ids },
  })
}

export function applySyncToCloud<T>(ids: number[]) {
  return post<T>({
    url: '/file/applySync',
    data: { ids },
  })
}

export function uploadToCloud<T>(ids: number[]) {
  return post<T>({
    url: '/file/uploadToCloud',
    data: { ids },
  })
}

export function getAuditList<T>() {
  return get<T>({
    url: '/file/auditList',
  })
}

export function auditAction<T>(ids: number[], action: 'approve' | 'reject') {
  return post<T>({
    url: '/file/auditAction',
    data: { ids, action },
  })
}

export function getPublicIcons<T>() {
  return get<T>({
    url: '/file/publicIcons',
  })
}

export function changeFileCategory<T>(ids: number[], fileType: number) {
  return post<T>({
    url: '/file/changeCategory',
    data: { ids, fileType },
  })
}

export function renameFile<T>(id: number, fileName: string) {
  return post<T>({
    url: '/file/rename',
    data: { id, fileName },
  })
}
