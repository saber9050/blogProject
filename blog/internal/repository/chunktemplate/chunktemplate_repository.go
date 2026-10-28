package chunktemplate

import (
	"errors"

	"blog/internal/model/entity"

	"gorm.io/gorm"
)

// chunkTable 向量块表名（与 entity.ArticleChunk.TableName() 一致）
const chunkTable = "article_chunks"

type chunkTemplateRepository struct {
	db *gorm.DB
}

// NewChunkTemplateRepository 创建切分模板仓储实例
func NewChunkTemplateRepository(db *gorm.DB) Repository {
	return &chunkTemplateRepository{db: db}
}

func (r *chunkTemplateRepository) List(status *int8) ([]*entity.ChunkTemplate, error) {
	var list []*entity.ChunkTemplate
	q := r.db.Order("id DESC")
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *chunkTemplateRepository) FindByID(id uint) (*entity.ChunkTemplate, error) {
	var t entity.ChunkTemplate
	err := r.db.First(&t, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *chunkTemplateRepository) ExistsByName(name string, excludeID uint) (bool, error) {
	var n int64
	q := r.db.Model(&entity.ChunkTemplate{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *chunkTemplateRepository) Create(t *entity.ChunkTemplate) error {
	return r.db.Create(t).Error
}

func (r *chunkTemplateRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&entity.ChunkTemplate{}).Where("id = ?", id).Updates(fields).Error
}

func (r *chunkTemplateRepository) Delete(id uint) error {
	return r.db.Delete(&entity.ChunkTemplate{}, id).Error
}

// Stats 按模板聚合。
//
// article_chunks 不存在时返回空 map 而不是错误：向量表是按需创建的，
// 用户可能还没配向量模型就要看模板列表，那时不该因为「表不存在」整页报错。
func (r *chunkTemplateRepository) Stats() (map[uint]TemplateStat, error) {
	out := make(map[uint]TemplateStat)

	ok, err := r.chunkTableExists()
	if err != nil {
		return nil, err
	}
	if !ok {
		return out, nil
	}

	var rows []struct {
		TemplateID   uint
		ArticleCount int64
		ChunkCount   int64
	}
	// count(DISTINCT article_id)：一篇文章在该模板下的块数会被合并成 1，
	// 这样才算得出「多少篇文章用过这个模板」
	err = r.db.Raw(`
		SELECT template_id,
		       count(DISTINCT article_id) AS article_count,
		       count(*)                   AS chunk_count
		FROM ` + chunkTable + `
		GROUP BY template_id`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TemplateID] = TemplateStat{
			ArticleCount: row.ArticleCount,
			ChunkCount:   row.ChunkCount,
		}
	}
	return out, nil
}

func (r *chunkTemplateRepository) ChunkCount(templateID uint) (int64, error) {
	ok, err := r.chunkTableExists()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	var n int64
	if err := r.db.Table(chunkTable).Where("template_id = ?", templateID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// chunkTableExists 向量表是否存在。
//
// 用 to_regclass 而不是查询 information_schema：前者是 O(1) 且不依赖 schema 名，
// 后者在 PG 里要扫系统表，这里每次列表请求都会调，不值得。
func (r *chunkTemplateRepository) chunkTableExists() (bool, error) {
	var name *string
	if err := r.db.Raw("SELECT to_regclass(?)::text", chunkTable).Scan(&name).Error; err != nil {
		return false, err
	}
	return name != nil, nil
}
