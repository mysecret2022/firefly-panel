import { get, post } from '@/utils/request'

export interface GitHubConfig {
  owner: string
  repo: string
  branch: string
  token: string
  basePath: string
  useCDN: boolean
}

export interface R2Config {
  accountId: string
  accessKeyId: string
  secretAccessKey: string
  bucketName: string
  publicDomain: string
}

export interface CloudStorageConfig {
  type: 'github' | 'r2'
  enabled: boolean
  github: GitHubConfig
  cloudflareR2: R2Config
}

export interface CloudFileItem {
  name: string
  path: string
  size: number
  downloadUrl: string
  updatedAt: string
  isDirectory: boolean
}

export function getCloudConfig() {
  return get<CloudStorageConfig>({
    url: '/panel/storage/getConfig',
  })
}

export function saveCloudConfig(data: CloudStorageConfig) {
  return post<{ message?: string }>({
    url: '/panel/storage/saveConfig',
    data,
  })
}

export function testCloudConnection(data?: CloudStorageConfig) {
  return post<{ isPrivate: boolean; description: string; message: string }>({
    url: '/panel/storage/test',
    data,
  })
}

export function uploadEncryptedBackup(password: string) {
  return post<{ filename: string; downloadUrl: string; size: number; message: string }>({
    url: '/panel/storage/uploadBackup',
    data: { password },
  })
}

export function restoreEncryptedBackup(filename: string, password: string) {
  return post<{ message: string }>({
    url: '/panel/storage/restoreBackup',
    data: { filename, password },
  })
}

export function listCloudBackups() {
  return get<CloudFileItem[]>({
    url: '/panel/storage/listBackups',
  })
}

export function listCloudImages() {
  return get<CloudFileItem[]>({
    url: '/panel/storage/listImages',
  })
}

export function syncImageToLocal(filename: string) {
  return post<{ localPath: string; filename: string; message: string }>({
    url: '/panel/storage/syncImage',
    data: { filename },
  })
}

export function syncAllImagesToLocal() {
  return post<{ total: number; syncedCount: number; message: string }>({
    url: '/panel/storage/syncAllImages',
  })
}

export function deleteCloudImages(paths: string[]) {
  return post<{ successCount: number; failCount: number; message: string }>({
    url: '/panel/storage/deleteImages',
    data: { paths },
  })
}

export function clearCloudImages() {
  return post<{ deletedCount: number; message: string }>({
    url: '/panel/storage/clearImages',
  })
}
