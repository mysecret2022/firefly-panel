package models

type File struct {
	BaseModel
	Src         string `json:"src"`
	UserId      uint   `json:"userId"`
	FileName    string `json:"fileName" gorm:"varchar(255)"` // 文件名
	Method      int    `gorm:"int(5)" json:"method"`         // 上传方式
	Ext         string `gorm:"varchar(255)" json:"ext"`      // 扩展名
	FileType    int    `gorm:"default:0" json:"fileType"`    // 0: 图库(壁纸/大图), 1: 应用图标
	SyncStatus  int    `gorm:"default:0" json:"syncStatus"`  // 0: 仅本地, 1: 待管理员审核, 2: 已同步云端, 3: 已驳回
	AuditRemark string `gorm:"varchar(255)" json:"auditRemark"`
}

// 添加一个文件记录
func (m *File) AddFile(userId uint, fileName, ext, src string) (File, error) {
	return m.AddFileWithCategory(userId, fileName, ext, src, 0)
}

// 添加一个带分类的文件记录
func (m *File) AddFileWithCategory(userId uint, fileName, ext, src string, fileType int) (File, error) {
	file := File{
		UserId:     userId,
		FileName:   fileName,
		Src:        src,
		Ext:        ext,
		FileType:   fileType,
		SyncStatus: 0,
	}
	err := Db.Create(&file).Error

	return file, err
}
